use std::process::{Child, Command, Stdio};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;
use std::sync::Mutex;

use tauri::State;

// Global backend process state
struct BackendState {
    process: Option<Child>,
    running: Arc<AtomicBool>,
    port: u16,
}

struct AppState {
    backend: Arc<Mutex<BackendState>>,
}

fn check_port_available(port: u16) -> bool {
    std::net::TcpListener::bind(format!("127.0.0.1:{}", port)).is_ok()
}

fn find_project_root() -> std::path::PathBuf {
    // Start from current directory and look for go_backend
    let mut path = std::env::current_dir().unwrap_or_default();

    // Try current dir first
    if path.join("go_backend").join("dashboard-backend").exists() {
        return path;
    }

    // Try parent (in case we're in src-tauri)
    if path.join("..").join("go_backend").join("dashboard-backend").exists() {
        return path.join("..");
    }

    // Walk up the directory tree
    loop {
        if path.join("go_backend").join("dashboard-backend").exists() {
            return path.clone();
        }
        if !path.pop() {
            break;
        }
    }

    // Fallback to current dir
    std::env::current_dir().unwrap_or_default()
}

fn get_backend_path() -> std::path::PathBuf {
    let root = find_project_root();
    let backend_path = root.join("go_backend").join("dashboard-backend");
    println!("[Rust] Looking for backend at: {:?}", backend_path);
    backend_path
}

fn start_go_backend(port: u16) -> Result<Child, String> {
    let backend_path = get_backend_path();

    if !backend_path.exists() {
        return Err(format!("Backend not found at {:?}", backend_path));
    }

    let child = Command::new(&backend_path)
        .env("DASHBOARD_PORT", port.to_string())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("Failed to spawn Go backend: {}", e))?;

    Ok(child)
}

fn wait_for_backend_ready(port: u16, timeout_secs: u64) -> bool {
    let start = std::time::Instant::now();
    while start.elapsed().as_secs() < timeout_secs {
        if let Ok(response) = ureq::get(&format!("http://127.0.0.1:{}/health", port))
            .timeout(Duration::from_secs(1))
            .call()
        {
            if response.status() == 200 {
                println!("[Rust] Backend ready on port {}", port);
                return true;
            }
        }
        std::thread::sleep(Duration::from_millis(200));
    }
    false
}

#[tauri::command]
fn get_backend_status(state: State<AppState>) -> Result<serde_json::Value, String> {
    let backend = state.backend.lock().unwrap();

    let pid = backend.process
        .as_ref()
        .map(|p| p.id())
        .unwrap_or(0);

    let running = backend.running.load(Ordering::SeqCst);

    let healthy = if running && pid > 0 {
        wait_for_backend_ready(backend.port, 1)
    } else {
        false
    };

    Ok(serde_json::json!({
        "running": running && healthy,
        "pid": pid,
        "port": backend.port,
        "healthy": healthy,
    }))
}

#[tauri::command]
fn restart_backend(state: State<AppState>) -> Result<serde_json::Value, String> {
    println!("[Rust] Restarting backend...");

    let port = {
        let mut backend = state.backend.lock().unwrap();

        if let Some(mut child) = backend.process.take() {
            let _ = child.kill();
            let _ = child.wait();
        }

        let child = start_go_backend(backend.port)?;
        backend.process = Some(child);
        backend.running.store(true, Ordering::SeqCst);

        backend.port
    };

    if wait_for_backend_ready(port, 10) {
        Ok(serde_json::json!({"status": "restarted", "port": port}))
    } else {
        Err("Backend restarted but not responding".to_string())
    }
}

#[tauri::command]
fn get_backend_url(state: State<AppState>) -> Result<String, String> {
    let backend = state.backend.lock().unwrap();
    Ok(format!("http://127.0.0.1:{}/", backend.port))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let backend_port = 18788u16;

    if !check_port_available(backend_port) {
        println!("[Rust] Warning: Port {} appears to be in use", backend_port);
    }

    // Pre-spawn backend for setup
    let backend_state = Arc::new(Mutex::new(BackendState {
        process: None,
        running: Arc::new(AtomicBool::new(false)),
        port: backend_port,
    }));

    match start_go_backend(backend_port) {
        Ok(child) => {
            let mut state = backend_state.lock().unwrap();
            state.process = Some(child);
            state.running.store(true, Ordering::SeqCst);
            println!("[Rust] Backend process spawned");
        }
        Err(e) => {
            eprintln!("[Rust] Failed to start backend: {}", e);
        }
    }

    // Wait for backend to be ready
    if wait_for_backend_ready(backend_port, 10) {
        println!("[Rust] Backend ready");
    }

    let backend_for_state = Arc::clone(&backend_state);

    tauri::Builder::default()
        .manage(AppState {
            backend: backend_for_state,
        })
        .setup(|_app| {
            println!("[Rust] Setup complete");
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            get_backend_status,
            restart_backend,
            get_backend_url,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

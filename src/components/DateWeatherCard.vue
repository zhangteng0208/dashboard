<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SolarDay, LunarHour, LunarYear, LunarMonth, God, Taboo, LegalHoliday } from 'tyme4ts'
import clearDaySvg from '/Users/mac/Documents/天气/clear-day.svg?raw'
import partlyCloudyDaySvg from '/Users/mac/Documents/天气/partly-cloudy-day.svg?raw'
import cloudySvg from '/Users/mac/Documents/天气/cloudy.svg?raw'
import overcastSvg from '/Users/mac/Documents/天气/overcast.svg?raw'
import fogSvg from '/Users/mac/Documents/天气/fog.svg?raw'
import rainSvg from '/Users/mac/Documents/天气/rain.svg?raw'
import drizzleSvg from '/Users/mac/Documents/天气/drizzle.svg?raw'
import snowSvg from '/Users/mac/Documents/天气/snow.svg?raw'
import thunderstormsSvg from '/Users/mac/Documents/天气/thunderstorms-day.svg?raw'
import hazeSvg from '/Users/mac/Documents/天气/haze.svg?raw'
import sleetSvg from '/Users/mac/Documents/天气/sleet.svg?raw'

const weather = ref<any>(null)
const weatherIconSvg = ref('')
const tomorrowIconSvg = ref('')
const showWeatherPopup = ref(false)
const forecastDays = ref<any[]>([])

async function fetchWeather() {
  try {
    // 使用 wttr.in 免费天气 API - 获取沈阳
    const res = await fetch('https://wttr.in/Shenyang?format=j1&lang=zh')
    if (res.ok) {
      const d = await res.json()
      const current = d.current_condition[0]
      const zhMap: Record<string, string> = {
        'Sunny': '晴天', 'Clear': '晴', 'Partly cloudy': '多云', 'Cloudy': '阴',
        'Overcast': '阴天', 'Mist': '薄雾', 'Fog': '雾', 'Light rain': '小雨',
        'Moderate rain': '中雨', 'Heavy rain': '大雨', 'Light snow': '小雪',
        'Moderate snow': '中雪', 'Heavy snow': '大雪', 'Thunderstorm': '雷暴',
        'Light drizzle': '毛毛雨', 'Patchy rain possible': '局部有雨',
        'Blowing snow': '吹雪', 'Blizzard': '暴雪', 'Freezing drizzle': '冻雨',
        'Freezing rain': '冰雨', 'Patchy light drizzle': '局部小雨',
        'Patchy moderate rain': '局部中雨', 'Moderate or heavy rain shower': '中到大雨',
        'Light rain shower': '小雨阵', 'Heavy rain shower': '大雨阵',
      }
      const wmoCode = current.weatherCode[0].value
      weatherIconSvg.value = getWeatherIconSvg(wmoCode)
      const tomorrowCode = d.weather?.[1]?.hourly?.[4]?.weatherCode?.[0]?.value || 'Sunny'
      tomorrowIconSvg.value = getWeatherIconSvg(tomorrowCode)
      weather.value = {
        temp: current.temp_C + '°C',
        tempMax: d.weather?.[0]?.maxtempC + '°' || '',
        tempMin: d.weather?.[0]?.mintempC + '°' || '',
        feelsLike: current.FeelsLikeC + '°C',
        condition: zhMap[wmoCode] || wmoCode,
        conditionEn: current.weatherDesc[0].value,
        humidity: current.humidity + '%',
        wind: current.windspeedKmph + ' km/h',
        windDir: translateWindDir(current.winddir16Point),
        pressure: current.pressure + ' hPa',
        visibility: current.visibility + ' km',
        sunrise: d.weather?.[0]?.astronomy?.[0]?.sunrise || '',
        sunset: d.weather?.[0]?.astronomy?.[0]?.sunset || '',
        tomorrow: d.weather?.[1] ? {
          tempMax: d.weather[1].maxtempC + '°',
          tempMin: d.weather[1].mintempC + '°',
          condition: zhMap[d.weather[1].hourly[4].weatherCode[0].value] || zhMap[d.weather[1].hourly[4].weatherDesc[0].value] || d.weather[1].hourly[4].weatherDesc[0].value,
        } : null,
      }
      // 3天预报
      forecastDays.value = d.weather?.map((w: any, idx: number) => ({
        day: idx === 0 ? '今日' : idx === 1 ? '明日' : '后日',
        date: w.date,
        tempMax: w.maxtempC + '°',
        tempMin: w.mintempC + '°',
        condition: zhMap[w.hourly?.[4]?.weatherCode?.[0]?.value] || zhMap[w.hourly?.[4]?.weatherDesc?.[0]?.value] || '',
        icon: getWeatherIconSvg(w.hourly?.[4]?.weatherCode?.[0]?.value || 'Sunny'),
      })) || []
    }
  } catch (e) {
    console.error('Failed to fetch weather:', e)
  }
}

function getWeatherIconSvg(code: string): string {
  const iconMap: Record<string, string> = {
    'Sunny': clearDaySvg,
    'Clear': clearDaySvg,
    'Partly cloudy': partlyCloudyDaySvg,
    'Cloudy': cloudySvg,
    'Overcast': overcastSvg,
    'Mist': fogSvg,
    'Fog': fogSvg,
    'Light rain': rainSvg,
    'Moderate rain': rainSvg,
    'Heavy rain': rainSvg,
    'Light snow': snowSvg,
    'Moderate snow': snowSvg,
    'Heavy snow': snowSvg,
    'Thunderstorm': thunderstormsSvg,
    'Light drizzle': drizzleSvg,
    'Patchy rain possible': drizzleSvg,
    'Blowing snow': snowSvg,
    'Blizzard': snowSvg,
    'Freezing drizzle': sleetSvg,
    'Freezing rain': rainSvg,
    'Patchy light drizzle': drizzleSvg,
    'Patchy moderate rain': rainSvg,
    'Moderate or heavy rain shower': rainSvg,
    'Light rain shower': rainSvg,
    'Heavy rain shower': rainSvg,
    'Haze': hazeSvg,
    'Smoke': hazeSvg,
  }
  return iconMap[code] || clearDaySvg
}

function translateWindDir(dir: string): string {
  const dirMap: Record<string, string> = {
    'N': '北风', 'NNE': '东北偏北', 'NE': '东北风', 'ENE': '东北偏东',
    'E': '东风', 'ESE': '东南偏东', 'SE': '东南风', 'SSE': '东南偏南',
    'S': '南风', 'SSW': '西南偏南', 'SW': '西南风', 'WSW': '西南偏西',
    'W': '西风', 'WNW': '西北偏西', 'NW': '西北风', 'NNW': '西北偏北',
  }
  return dirMap[dir] || dir
}

const data = ref<any>(null)
const weekdays = ['日', '一', '二', '三', '四', '五', '六']
const currentTime = ref('')
const hourNames = ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥']
const sixStarsNames = ['大安', '友引', '先胜', '先负', '赤口', '佛灭']
const twelveStarNames = ['青龙', '明堂', '天刑', '朱雀', '金匮', '天德', '白虎', '玉堂', '天牢', '玄武', '司命', '勾陈']
const calYear = ref(new Date().getFullYear())
const calMonth = ref(new Date().getMonth() + 1)
const solarTermsData = ['小寒', '大寒', '立春', '雨水', '惊蛰', '春分', '清明', '谷雨', '立夏', '小满', '芒种', '夏至', '小暑', '大暑', '立秋', '处暑', '白露', '秋分', '寒露', '霜降', '立冬', '小雪', '大雪', '冬至']
const wuHouData: Record<string, string> = {
  '小寒': '雁北乡|鹊始巢|雉始雊', '大寒': '鸡始乳|征鸟厉疾|水泽腹坚',
  '立春': '东风解冻|蛰虫始振|鱼陟负冰', '雨水': '獭祭鱼|鸿雁来|草木萌动',
  '惊蛰': '桃始华|仓庚鸣|鹰化为鸠', '春分': '玄鸟至|雷乃发声|始电',
  '清明': '桐始华|田鼠化为鴽|虹始见', '谷雨': '萍始生|鸣鸠拂其羽|戴胜降于桑',
  '立夏': '蝼蝈鸣|蚯蚓出|王瓜生', '小满': '苦菜秀|靡草死|麦秋至',
  '芒种': '螳螂生|鵙始鸣|反舌无声', '夏至': '鹿角解|蜩始鸣|半夏生',
  '小暑': '温风至|蟋蟀居壁|鹰始挚', '大暑': '腐草为萤|土润溽暑|大雨时行',
  '立秋': '凉风至|白露生|寒蝉鸣', '处暑': '鹰乃祭鸟|天地始肃|禾乃登',
  '白露': '鸿雁来|玄鸟归|群鸟养羞', '秋分': '雷始收声|蛰虫坯户|水始涸',
  '寒露': '鸿雁来宾|雀入大水为蛤|菊有黄华', '霜降': '豺乃祭兽|草木黄落|蛰虫咸俯',
  '立冬': '水始冰|地始冻|雉入大水为蜃', '小雪': '虹藏不见|天气上升|闭塞成冬',
  '大雪': '鹖鴠不鸣|虎始交|荔挺出', '冬至': '蚯蚓结|麋角解|水泉动',
}
const nineStarNames = ['一白水', '二黑土', '三碧木', '四绿木', '五黄土', '六白金', '七赤金', '八白土', '九紫火']

function getLuckClass(name: string): string {
  if (!name) return ''
  if (name.includes('吉') || name.includes('贵') || name.includes('福') || name.includes('德') || name.includes('喜') || name.includes('神')) return 'good'
  if (name.includes('凶') || name.includes('煞') || name.includes('劫') || name.includes('死') || name.includes('破')) return 'bad'
  return ''
}

function getElementClass(name: string): string {
  if (!name) return ''
  if (name.includes('金')) return 'el-gold'
  if (name.includes('木')) return 'el-wood'
  if (name.includes('水')) return 'el-water'
  if (name.includes('火')) return 'el-fire'
  if (name.includes('土')) return 'el-earth'
  return ''
}


function getHourLuckList(currentShichen: number) {
  return hourNames.map((n, idx) => ({
    name: n,
    luck: idx === currentShichen ? '凶' : (idx % 3 === 0 ? '凶' : '吉'),
    isCurrent: idx === currentShichen
  }))
}

function updateTime() {
  const now = new Date()
  const h = now.getHours().toString().padStart(2, '0')
  const m = now.getMinutes().toString().padStart(2, '0')
  const s = now.getSeconds().toString().padStart(2, '0')
  currentTime.value = `${h}:${m}:${s}`

  // Update hour luck highlighting (时辰: 0-1点=子, 2-3点=丑, ...)
  if (data.value) {
    const currentHour = now.getHours()
    const currentShichen = Math.floor(currentHour / 2) % 12
    data.value.hourLuck = getHourLuckList(currentShichen)
  }
}

function prevMonth() {
  if (calMonth.value === 1) {
    calMonth.value = 12
    calYear.value--
  } else {
    calMonth.value--
  }
  updateCalendar()
}

function nextMonth() {
  if (calMonth.value === 12) {
    calMonth.value = 1
    calYear.value++
  } else {
    calMonth.value++
  }
  updateCalendar()
}

function updateCalendar() {
  const year = calYear.value
  const month = calMonth.value
  const day = new Date().getDate()

  const calDays: any[] = []
  const firstDay = new Date(year, month - 1, 1)
  const lastDay = new Date(year, month, 0)
  const startWeekday = firstDay.getDay()
  const daysInMonth = lastDay.getDate()

  for (let i = 0; i < startWeekday; i++) {
    calDays.push({ empty: true })
  }
  for (let d = 1; d <= daysInMonth; d++) {
    const sd = SolarDay.fromYmd(year, month, d)
    const ld = sd.getLunarDay()
    const lf = ld.getFestival()
    const sf = sd.getFestival()
    const lh2 = LegalHoliday.fromYmd(year, month, d)
    calDays.push({
      day: d,
      isToday: d === day && month === new Date().getMonth() + 1 && year === new Date().getFullYear(),
      lunar: ld.getLunarMonth().getName() + ld.getName(),
      lunarShort: ld.getName(),
      festival: lf?.getName() || sf?.getName() || '',
      legal: lh2?.getName() || '',
      isWork: lh2?.isWork() || false,
    })
  }

  data.value.calendarDays = calDays
  data.value.calendarMonth = `${year}年${month}月`
}

function init() {
  try {
    const now = new Date()
    const year = now.getFullYear()
    const month = now.getMonth() + 1
    const day = now.getDate()
    const hour = now.getHours()

    const solar = SolarDay.fromYmd(year, month, day)
    const lunar = solar.getLunarDay()

    let lh: any, ec: any
    try {
      lh = LunarHour.fromYmdHms(year, month, day, hour.toString(), '0', '0')
      ec = lh.getEightChar()
    } catch {
      lh = null
      ec = null
    }

    const yearSixty = lunar.getYearSixtyCycle()
    const monthSixty = lunar.getMonthSixtyCycle()
    const daySixty = lunar.getSixtyCycle()
    const duty = lunar.getDuty()
    const twentyEightStar = lunar.getTwentyEightStar()
    const nineStar = lunar.getNineStar()
    const sixStar = lunar.getSixStar()
    const phase = lunar.getPhase()
    const sevenStar = lunar.getWeek().getSevenStar()
    const land = twentyEightStar.getLand()
    const term = solar.getTerm()
    const scd = lunar.getSixtyCycleDay()
    const twelveStar = scd.getTwelveStar()
    const fetusDay = lunar.getFetusDay()
    const minorRen = lunar.getMinorRen()
    const dogDay = solar.getDogDay()
    const plumRainDay = solar.getPlumRainDay()
    const nineDay = solar.getNineDay()
    const hideHeavenStemDay = solar.getHideHeavenStemDay()
    const yearNineStar = LunarYear.fromYear(year).getNineStar()
    const monthNineStar = LunarMonth.fromYm(year, month).getNineStar()
    const hourNineStar = lh ? lh.getNineStar() : null
    const lunarYear = LunarYear.fromYear(year)
    const twenty = lunarYear.getTwenty()
    const yuan = twenty.getSixty()
    const xun = daySixty.getTen()

    const dayEB = daySixty.getEarthBranch()
    const dayHS = daySixty.getHeavenStem()
    const pengZu = daySixty.getPengZu()
    const jupiterDir = lunar.getJupiterDirection()

    const sixStarIndex = sixStar ? sixStar.getIndex() : -1
    const termIdx = term ? solarTermsData.indexOf(term.getName()) : -1

    const lunarDayNum = lunar.getDay()
    const buddhistEvents: string[] = []
    if ([1, 8, 14, 15, 23, 29].includes(lunarDayNum)) buddhistEvents.push('六斋')
    if ([1, 8, 14, 15, 18, 23, 24, 28, 29, 30].includes(lunarDayNum)) buddhistEvents.push('十斋')
    if ([1, 8, 15].includes(lunarDayNum)) buddhistEvents.push('观音')
    if (lunarDayNum === 13) buddhistEvents.push('杨忌')

    const currentShichen = Math.floor(hour / 2) % 12
    const hourLuckList = getHourLuckList(currentShichen)

    // Calendar data
    const calDays: any[] = []
    const firstDay = new Date(year, month - 1, 1)
    const lastDay = new Date(year, month, 0)
    const startWeekday = firstDay.getDay()
    const daysInMonth = lastDay.getDate()

    for (let i = 0; i < startWeekday; i++) {
      calDays.push({ empty: true })
    }
    for (let d = 1; d <= daysInMonth; d++) {
      const sd = SolarDay.fromYmd(year, month, d)
      const ld = sd.getLunarDay()
      const lf = ld.getFestival()
      const sf = sd.getFestival()
      const lh2 = LegalHoliday.fromYmd(year, month, d)
      calDays.push({
        day: d,
        isToday: d === day,
        lunar: ld.getLunarMonth().getName() + ld.getName(),
        lunarShort: ld.getName(),
        festival: lf?.getName() || sf?.getName() || '',
        legal: lh2?.getName() || '',
        isWork: lh2?.isWork() || false,
      })
    }

    let yrP: any, moP: any, dyP: any, hrP: any
    let pillarTianGan: any[] = [], pillarDiZhi: any[] = []
    if (ec) {
      yrP = ec.getYear()
      moP = ec.getMonth()
      dyP = ec.getDay()
      hrP = ec.getHour()
      pillarTianGan = [yrP, moP, dyP, hrP]
      pillarDiZhi = [yrP.getEarthBranch(), moP.getEarthBranch(), dyP.getEarthBranch(), hrP.getEarthBranch()]
    }

    const allGods = lunar.getGods()
    const goodGods = allGods.filter((g: God) => g.getIndex() < 60).map((g: God) => g.getName())
    const badGods = allGods.filter((g: God) => g.getIndex() >= 60).map((g: God) => g.getName())

    data.value = {
      solar: { date: `${year}年${month}月${day}日`, weekday: weekdays[now.getDay()], year, month, day },
      lunar: {
        name: lunar.getName(), monthName: lunar.getLunarMonth().getName(),
        isLeap: lunar.getLunarMonth().isLeap(), yearName: yearSixty.getName(),
        animal: yearSixty.getEarthBranch().getZodiac().getName(),
      },
      ganzhi: {
        year: yearSixty.getName(), month: monthSixty.getName(),
        day: daySixty.getName(), hour: lh ? lh.getSixtyCycle().getName() : '',
      },
      clash: ec ? {
        zodiac: dayEB.getZodiac().getName(), direction: dayEB.getDirection().getName(),
        stem: dayHS.getName(), element: dayEB.getElement().getName(), naYin: daySixty.getSound().getName(),
      } : { zodiac: '', direction: '', stem: '', element: '', naYin: '' },
      duty: { name: duty ? duty.getName() : '' },
      tenStarDay: ec ? dayHS.getTenStar(dyP.getHeavenStem()).getName() : '',
      terrainDay: ec ? dyP.getHeavenStem().getTerrain(dyP.getEarthBranch()).getName() : '',
      pillarTenStar: ec ? pillarTianGan.map(tg => dayHS.getTenStar(tg.getHeavenStem()).getName()) : [],
      pillarTerrain: ec ? pillarTianGan.map((tg, i) => tg.getHeavenStem().getTerrain(pillarDiZhi[i]).getName()) : [],
      twelveStar: { name: twelveStar ? twelveStar.getName() : '' },
      sixStar: { name: sixStar ? sixStar.getName() : '' },
      phase: { name: phase ? phase.getName() : '' },
      twentyEightStar: {
        name: twentyEightStar ? twentyEightStar.getName() : '',
        sevenStar: twentyEightStar ? twentyEightStar.getSevenStar().getName() : '',
        animal: twentyEightStar ? twentyEightStar.getAnimal().getName() : '',
        zone: twentyEightStar ? twentyEightStar.getZone().getName() : '',
        land: twentyEightStar ? twentyEightStar.getLand().getName() : '',
      },
      nineStar: {
        name: nineStar ? nineStar.getName() : '',
        colorName: nineStar ? nineStarNames[nineStar.getIndex()] : '',
        dipper: nineStar ? nineStar.getDipper().getName() : '',
        direction: nineStar ? nineStar.getDirection().getName() : '',
      },
      jieQi: { name: term ? term.getName() : '', wuHou: wuHouData[term ? term.getName() : ''] || '' },
      festival: {
        lunar: lunar.getFestival()?.getName() || '',
        solar: solar.getFestival()?.getName() || '',
        legal: solar.getLegalHoliday()?.getName() || '',
        isWork: solar.getLegalHoliday()?.isWork() || false,
      },
      constellation: solar.getConstellation().getName(),
      fetusDay: {
        stem: fetusDay ? fetusDay.getFetusHeavenStem().getName() : '',
        branch: fetusDay ? fetusDay.getFetusEarthBranch().getName() : '',
        direction: fetusDay ? fetusDay.getDirection().getName() : '',
      },
      pengZu: { bans: pengZu ? pengZu.getName() : '' },
      jupiterDirection: jupiterDir ? jupiterDir.getName() : '',
      minorRen: minorRen ? minorRen.getName() : '',
      dogDay: dogDay ? `数九第${(dogDay as any).getDayCount?.() || 0}天` : '',
      plumRain: plumRainDay ? `${plumRainDay.getName()}` : '',
      nineDay: nineDay ? nineDay.getName() : '',
      hideHeavenStem: hideHeavenStemDay ? hideHeavenStemDay.getName() : '',
      recommends: lunar.getRecommends().map((t: Taboo) => t.getName()),
      avoids: lunar.getAvoids().map((t: Taboo) => t.getName()),
      goodGods, badGods, buddhistEvents, hourLuck: hourLuckList, hour,
      eightChar: ec ? {
        year: yrP.getName(), month: moP.getName(), day: dyP.getName(), hour: hrP.getName(),
        tianGan: { year: yrP.getHeavenStem().getName(), month: moP.getHeavenStem().getName(), day: dyP.getHeavenStem().getName(), hour: hrP.getHeavenStem().getName() },
        diZhi: { year: yrP.getEarthBranch().getName(), month: moP.getEarthBranch().getName(), day: dyP.getEarthBranch().getName(), hour: hrP.getEarthBranch().getName() },
        naYin: { year: yrP.getSound().getName(), month: moP.getSound().getName(), day: dyP.getSound().getName(), hour: hrP.getSound().getName() },
        yinYang: { year: yrP.getEarthBranch().getYinYang() === 0 ? '阴' : '阳', month: moP.getEarthBranch().getYinYang() === 0 ? '阴' : '阳', day: dyP.getEarthBranch().getYinYang() === 0 ? '阴' : '阳', hour: hrP.getEarthBranch().getYinYang() === 0 ? '阴' : '阳' },
        wuXing: { year: yrP.getEarthBranch().getElement().getName(), month: moP.getEarthBranch().getElement().getName(), day: dyP.getEarthBranch().getElement().getName(), hour: hrP.getEarthBranch().getElement().getName() },
        shiShen: { year: yrP.getTen().getName(), month: moP.getTen().getName(), day: dyP.getTen().getName(), hour: hrP.getTen().getName() },
        pengZu: { year: yrP.getPengZu().getName(), month: moP.getPengZu().getName(), day: dyP.getPengZu().getName(), hour: hrP.getPengZu().getName() },
        zangGan: {
          year: yrP.getEarthBranch().getHideHeavenStems().map((h: any) => h.getName()).join('·'),
          month: moP.getEarthBranch().getHideHeavenStems().map((h: any) => h.getName()).join('·'),
          day: dyP.getEarthBranch().getHideHeavenStems().map((h: any) => h.getName()).join('·'),
          hour: hrP.getEarthBranch().getHideHeavenStems().map((h: any) => h.getName()).join('·'),
        },
        taiYuan: ec.getFetalOrigin().getName(), taiXi: ec.getFetalBreath().getName(),
        shenGong: ec.getOwnSign().getName(), mingGong: ec.getBodySign().getName(),
        yongShen: ec.getDuty().getName(),
      } : null,
      sixStars: sixStarsNames.map((name, idx) => ({ name, active: idx === sixStarIndex })),
      sevenStar: sevenStar ? { name: sevenStar.getName(), week: sevenStar.getWeek().getName() } : { name: '', week: '' },
      land: land ? { name: land.getName(), direction: land.getDirection().getName() } : { name: '', direction: '' },
      nineStarDisplay: nineStar ? { name: nineStar.getName(), color: nineStar.getColor(), element: nineStar.getElement().getName(), dipper: nineStar.getDipper().getName() } : { name: '', color: '', element: '', dipper: '' },
      pillarNineStar: {
        year: { name: nineStarNames[yearNineStar.getIndex()] },
        month: { name: nineStarNames[monthNineStar.getIndex()] },
        day: { name: nineStar ? nineStarNames[nineStar.getIndex()] : '' },
        hour: { name: hourNineStar ? nineStarNames[hourNineStar.getIndex()] : '' },
      },
      twelveStars: twelveStarNames.map((name, idx) => ({ name, active: idx === (twelveStar ? twelveStar.getIndex() : -1) })),
      solarTerms: solarTermsData.map((name, idx) => ({ name, active: idx === termIdx, past: idx < termIdx })),
      yun: { name: twenty.getName() },
      yuan: { name: yuan.getName() },
      xun: { name: xun.getName() },
      calendarDays: calDays,
      calendarMonth: `${year}年${month}月`,
    }
  } catch (e) {
    console.error('Failed to init lunar data:', e)
  }
}

onMounted(() => { init(); updateTime(); fetchWeather(); setInterval(updateTime, 1000) })
</script>

<template>
  <div class="cal" v-if="data">
    <!-- ═══ OUTER BORDER FRAME ═══ -->
    <div class="frame-outer">
      <div class="frame-inner">

        <!-- ─── WEATHER ROW ─── -->
        <div class="weather-row" v-if="weather">
          <div class="weather-main-block">
            <div class="weather-icon-lg" v-html="weatherIconSvg"></div>
            <div class="weather-primary">
              <span class="weather-temp">{{ weather.temp }}</span>
              <span class="weather-feels">体感 {{ weather.feelsLike }}</span>
            </div>
            <div class="weather-condition-block">
              <div class="ws-item">
                <span class="ws-label">最高/最低</span>
                <span class="ws-val" style="white-space: nowrap">{{ weather.tempMin }} ~ {{ weather.tempMax }}</span>
              </div>
              <div class="ws-item">
                <span class="ws-label">风向</span>
                <span class="ws-val">{{ weather.windDir }}</span>
              </div>
            </div>
          </div>
          <div class="weather-stats-grid">
            <div class="ws-item">
              <span class="ws-label">湿度</span>
              <span class="ws-val" style="white-space: nowrap">{{ weather.humidity }}</span>
            </div>
            <div class="ws-item">
              <span class="ws-label">风速</span>
              <span class="ws-val" style="white-space: nowrap">{{ weather.wind }}</span>
            </div>
            <div class="ws-item">
              <span class="ws-label">气压</span>
              <span class="ws-val" style="white-space: nowrap">{{ weather.pressure }}</span>
            </div>
            <div class="ws-item">
              <span class="ws-label">能见度</span>
              <span class="ws-val" style="white-space: nowrap">{{ weather.visibility }}</span>
            </div>
          </div>
          <div class="weather-sun-times">
            <div class="sun-item">
              <span class="sun-label">日出</span>
              <span class="sun-val" style="white-space: nowrap">{{ weather.sunrise }}</span>
            </div>
            <div class="sun-item">
              <span class="sun-label">日落</span>
              <span class="sun-val" style="white-space: nowrap">{{ weather.sunset }}</span>
            </div>
          </div>
          <div class="weather-tomorrow" v-if="weather.tomorrow" @click="showWeatherPopup = true">
            <span class="tomorrow-label">明日</span>
            <div class="tomorrow-icon" v-html="tomorrowIconSvg"></div>
            <span class="tomorrow-temp" style="white-space: nowrap">{{ weather.tomorrow.tempMin }} ~ {{ weather.tomorrow.tempMax }}</span>
          </div>
        </div>

        <!-- ─── WEATHER POPUP ─── -->
        <div class="weather-popup-overlay" v-if="showWeatherPopup" @click="showWeatherPopup = false">
          <div class="weather-popup" @click.stop>
            <div class="popup-header">
              <span class="popup-title">天气预报</span>
              <span class="popup-close" @click="showWeatherPopup = false">✕</span>
            </div>
            <div class="forecast-list">
              <div class="forecast-item" v-for="day in forecastDays" :key="day.date">
                <span class="forecast-day">{{ day.day }}</span>
                <div class="forecast-icon" v-html="day.icon"></div>
                <span class="forecast-condition">{{ day.condition }}</span>
                <span class="forecast-temp">{{ day.tempMin }} ~ {{ day.tempMax }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- ─── DATE ROW ─── -->
        <div class="date-row">
          <div class="date-cell solar-cell">
            <span class="dc-label">公历</span>
            <span class="dc-main">{{ data.solar.date }}</span>
            <span class="dc-sub">星期{{ data.solar.weekday }}</span>
            <span class="dc-time">{{ currentTime }}</span>
          </div>
          <div class="date-cell lunar-cell">
            <span class="dc-label">农历</span>
            <span class="l-year">{{ data.lunar.yearName }}年</span>
            <span class="dc-main">
              <span class="l-month" :class="{ leap: data.lunar.isLeap }">
                {{ data.lunar.isLeap ? '闰' : '' }}{{ data.lunar.monthName }}
              </span>
              <span class="l-day">{{ data.lunar.name }}</span>
            </span>
            <span class="dc-sub">{{ data.lunar.animal }}肖</span>
          </div>
          <div class="date-cell ganzhi-cell">
            <span class="dc-label">本日干支</span>
            <span class="dc-main gold">{{ data.ganzhi.day }}</span>
            <span class="dc-sub" :class="getElementClass(data.clash.naYin)">{{ data.clash.naYin }}</span>
          </div>
          <div class="date-cell jieqi-cell">
            <span class="dc-label">节气</span>
            <span class="dc-main gold">{{ data.jieQi.name || '日常' }}</span>
            <span class="dc-sub wuhou">{{ (data.jieQi.wuHou || '无候').replace(/\|/g, ' · ') }}</span>
          </div>
        </div>

        <!-- ─── GANZHI PILLARS ─── -->
        <div class="pillars-row" v-if="data.eightChar">
          <div class="pillar-block" v-for="(col, i) in ['year','month','day','hour']" :key="col"
            :class="{ 'pillar-highlight': col === 'day' }">
            <div class="pb-header">{{ ['年柱','月柱','日柱','时柱'][i] }}</div>
            <div class="pb-name">{{ data.ganzhi[col] }}</div>
            <div class="pb-stems">{{ data.eightChar?.tianGan?.[col] }} {{ data.eightChar?.diZhi?.[col] }}</div>
            <div class="pb-el" :class="getElementClass(data.eightChar?.wuXing?.[col])">{{ data.eightChar?.wuXing?.[col] }} · {{ data.eightChar?.yinYang?.[col] }}</div>
            <div class="pb-nine">{{ data.pillarNineStar[col].name }}</div>
          </div>
        </div>

        <!-- ─── YI/JI SPLIT (below pillars) ─── -->
        <div class="section s-yi">
          <div class="sec-header">
            <div class="sec-ornament"></div>
            <span class="sec-title">宜忌</span>
            <div class="sec-line"></div>
          </div>
          <div class="yi-ji-split">
            <div class="yj-half yj-yi">
              <div class="yj-chips">
                <span class="yj-chip yi-chip" v-for="r in data.recommends" :key="r">{{ r }}</span>
              </div>
            </div>
            <div class="yj-divider-v"></div>
            <div class="yj-half yj-ji">
              <div class="yj-chips">
                <span class="yj-chip ji-chip" v-for="a in data.avoids" :key="a">{{ a }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- ─── MAIN GRID ─── -->
        <div class="main-grid">
          <!-- LEFT: Hour + BaZi -->
          <div class="col-left">
            <!-- Hour Luck -->
            <div class="section s-hour">
              <div class="sec-header">
                <div class="sec-ornament"></div>
                <span class="sec-title">时辰吉凶</span>
                <span class="cur-hour-tag">本时: {{ hourNames[data.hour] }}</span>
                <div class="sec-line"></div>
              </div>
              <div class="hour-bar">
                <div v-for="h in data.hourLuck" :key="h.name"
                  class="hb-cell" :class="{ cur: h.isCurrent, good: h.luck === '吉', bad: h.luck === '凶' }">
                  <span class="hb-name">{{ h.name }}</span>
                  <div class="hb-dot"></div>
                  <span class="hb-luck">{{ h.luck }}</span>
                </div>
              </div>
            </div>

            <!-- Eight Characters Table -->
            <div class="section s-bazi" v-if="data.eightChar">
              <div class="sec-header">
                <div class="sec-ornament"></div>
                <span class="sec-title">四柱八字</span>
                <div class="sec-line"></div>
              </div>
              <div class="bazi-matrix">
                <div class="bm-row bm-head">
                  <span></span><span>年</span><span>月</span><span :class="{ gold: true }">日</span><span>时</span>
                </div>
                <div class="bm-row" v-for="row in [
                  {l:'干支', k:'year'},
                  {l:'天干', k:'tianGan'},
                  {l:'地支', k:'diZhi'},
                  {l:'纳音', k:'naYin', cl:'na'},
                  {l:'阴阳', k:'yinYang'},
                  {l:'五行', k:'wuXing', cl:'el'},
                  {l:'十神', k:'shiShen', cl:'shishen'},
                  {l:'彭祖', k:'pengZu', cl:'pz'},
                  {l:'藏干', k:'zangGan', cl:'zg'},
                ]" :key="row.l">
                  <span class="bm-lbl">{{ row.l }}</span>
                  <span v-for="col in ['year','month','day','hour']" :key="col"
                    class="bm-cell" :class="(row.k === 'wuXing' || row.k === 'naYin') ? getElementClass(data.eightChar?.[row.k]?.[col]) : row.cl">
                    {{ row.k === 'zangGan' ? data.eightChar?.zangGan?.[col] : data.eightChar?.[row.k]?.[col] }}
                  </span>
                </div>
              </div>
              <!-- Ming Table -->
              <div class="ming-table">
                <div class="mt-item"><span class="mt-lbl">胎元</span><span class="mt-val">{{ data.eightChar?.taiYuan }}</span></div>
                <div class="mt-item"><span class="mt-lbl">胎息</span><span class="mt-val">{{ data.eightChar?.taiXi }}</span></div>
                <div class="mt-item"><span class="mt-lbl">身宫</span><span class="mt-val">{{ data.eightChar?.shenGong }}</span></div>
                <div class="mt-item"><span class="mt-lbl">命宫</span><span class="mt-val">{{ data.eightChar?.mingGong }}</span></div>
                <div class="mt-item mt-highlight"><span class="mt-lbl">用神</span><span class="mt-val gold">{{ data.eightChar?.yongShen }}</span></div>
              </div>
            </div>
          </div>

          <!-- RIGHT: Gods -->
          <div class="col-right">
            <!-- Gods -->
            <div class="section s-gods">
              <div class="sec-header">
                <div class="sec-ornament"></div>
                <span class="sec-title">吉神凶煞</span>
                <div class="sec-line"></div>
              </div>
              <div class="gods-2col">
                <div class="god-col">
                  <div class="god-head">
                    <span class="god-dot god-dot-good"></span>
                    <span class="god-head-txt good-txt">吉神 {{ data.goodGods.length }}</span>
                  </div>
                  <div class="god-tags">
                    <span class="god-tag gt-good" v-for="g in data.goodGods" :key="g">{{ g }}</span>
                  </div>
                </div>
                <div class="god-col">
                  <div class="god-head">
                    <span class="god-dot god-dot-bad"></span>
                    <span class="god-head-txt bad-txt">凶煞 {{ data.badGods.length }}</span>
                  </div>
                  <div class="god-tags">
                    <span class="god-tag gt-bad" v-for="g in data.badGods" :key="g">{{ g }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ─── STARS + INFO ROW ─── -->
        <div class="stars-info-row">
          <div class="section s-stars">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">星宿神煞</span>
              <div class="sec-line"></div>
            </div>
            <div class="stars-row">
              <div class="star-card main-star">
                <div class="sc-key">二十八宿</div>
                <div class="sc-val lg" :class="getLuckClass(data.twentyEightStar.name)">{{ data.twentyEightStar.name }}</div>
                <div class="sc-sub">{{ data.twentyEightStar.animal }}/{{ data.twentyEightStar.zone }}/{{ data.twentyEightStar.land }}</div>
              </div>
              <div class="star-card gold-star">
                <div class="sc-key">九星</div>
                <div class="sc-val lg gold">{{ data.nineStar.colorName }}</div>
                <div class="sc-sub">{{ data.nineStar.dipper }} · {{ data.nineStar.direction }}</div>
              </div>
              <div class="star-card" v-for="s in [
                {k:'值神', v: data.duty.name, c: getLuckClass(data.duty.name)},
                {k:'十二值', v: data.twelveStar.name, c: ''},
                {k:'七曜', v: data.twentyEightStar.sevenStar, c: ''},
                {k:'月相', v: data.phase.name, c: ''},
                {k:'六曜', v: data.sixStar.name, c: getLuckClass(data.sixStar.name)},
                {k:'纳音', v: data.clash.naYin, c: getElementClass(data.clash.naYin)},
              ]" :key="s.k">
                <div class="sc-key">{{ s.k }}</div>
                <div class="sc-val" :class="s.c">{{ s.v }}</div>
              </div>
            </div>
          </div>
          <div class="section s-info">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">冲煞五行</span>
              <div class="sec-line"></div>
            </div>
            <div class="info-4grid">
              <div class="info-chip-row">
                <span class="icr-label">冲</span>
                <span class="icr-val danger">{{ data.clash.zodiac }}{{ data.clash.direction }}</span>
              </div>
              <div class="info-chip-row">
                <span class="icr-label">煞</span>
                <span class="icr-val danger">{{ data.clash.direction }}</span>
              </div>
              <div class="info-chip-row">
                <span class="icr-label">五行</span>
                <span class="icr-val" :class="getElementClass(data.clash.element)">{{ data.clash.element }}</span>
              </div>
              <div class="info-chip-row">
                <span class="icr-label">纳音</span>
                <span class="icr-val" :class="getElementClass(data.clash.naYin)">{{ data.clash.naYin }}</span>
              </div>
              <div class="info-chip-row">
                <span class="icr-label">财神</span>
                <span class="icr-val">{{ data.jupiterDirection }}</span>
              </div>
              <div class="info-chip-row">
                <span class="icr-label">吉门</span>
                <span class="icr-val">{{ data.nineStar.dipper }}门</span>
              </div>
              <div class="info-chip-row wide">
                <span class="icr-label">胎神</span>
                <span class="icr-val">{{ data.fetusDay.stem }}{{ data.fetusDay.branch }} {{ data.fetusDay.direction }}</span>
              </div>
              <div class="info-chip-row wide">
                <span class="icr-label">彭祖</span>
                <span class="icr-val sm">{{ data.pengZu.bans }}</span>
              </div>
              <div class="info-chip-row wide" v-if="data.buddhistEvents.length">
                <span class="icr-label">斋日</span>
                <div class="bud-tags">
                  <span class="bud-tag" v-for="e in data.buddhistEvents" :key="e">{{ e }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ─── BOTTOM ROW: Six Stars + SevenStar + Land + NineStar ─── -->
        <div class="bottom-row four-col">
          <div class="bottom-section six-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">六曜</span>
              <div class="sec-line"></div>
            </div>
            <div class="six-single">
              <span class="six-name">{{ data.sixStars.find((s: any) => s.active)?.name || '' }}</span>
            </div>
          </div>

          <div class="bottom-section seven-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">七曜</span>
              <div class="sec-line"></div>
            </div>
            <div class="seven-star">
              <span class="seven-name">{{ data.sevenStar.name }}</span>
              <span class="seven-sub">{{ data.sevenStar.week }}</span>
            </div>
          </div>

          <div class="bottom-section land-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">九野</span>
              <div class="sec-line"></div>
            </div>
            <div class="land-info">
              <span class="land-name">{{ data.land.name }}</span>
              <span class="land-sub">{{ data.land.direction }}</span>
            </div>
          </div>

          <div class="bottom-section nine-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">九星</span>
              <div class="sec-line"></div>
            </div>
            <div class="nine-star">
              <span class="nine-name">{{ data.nineStarDisplay.name }}</span>
              <span class="nine-sub">{{ data.nineStarDisplay.dipper }}</span>
            </div>
          </div>
        </div>

        <!-- ─── NEW ROW: TenStar + Terrain + Duty ─── -->
        <div class="bottom-row three-col">
          <div class="bottom-section tenstar-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">十神</span>
              <div class="sec-line"></div>
            </div>
            <div class="tenstar-display">
              <div class="tenstar-item" v-for="(col, i) in ['year','month','day','hour']" :key="col">
                <span class="ts-label">{{ ['年','月','日','时'][i] }}</span>
                <span class="ts-val">{{ data.pillarTenStar[i] }}</span>
              </div>
            </div>
          </div>

          <div class="bottom-section terrain-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">长生十二神</span>
              <div class="sec-line"></div>
            </div>
            <div class="terrain-display">
              <div class="terrain-item" v-for="(col, i) in ['year','month','day','hour']" :key="col">
                <span class="ts-label">{{ ['年','月','日','时'][i] }}</span>
                <span class="ts-val">{{ data.pillarTerrain[i] }}</span>
              </div>
            </div>
          </div>

          <div class="bottom-section duty-section">
            <div class="sec-header">
              <div class="sec-ornament"></div>
              <span class="sec-title">建除</span>
              <div class="sec-line"></div>
            </div>
            <div class="duty-single">
              <span class="duty-name">{{ data.duty.name }}</span>
            </div>
          </div>
        </div>

        <!-- ─── FOOTER: JieQi timeline ─── -->
        <div class="jieqi-footer">
          <div class="sec-header no-ornament">
            <span class="sec-title">二十四节气</span>
            <div class="sec-line"></div>
          </div>
          <div class="jq-track">
            <span v-for="t in data.solarTerms" :key="t.name"
              class="jq-item" :class="{ active: t.active, past: t.past }">{{ t.name }}</span>
          </div>
        </div>

        <!-- ─── EXTRA INFO ─── -->
        <div class="extra-section">
          <div class="extra-row">
            <div class="extra-cell" v-if="data.constellation">
              <span class="ec-label">星座</span>
              <span class="ec-val gold">{{ data.constellation }}</span>
            </div>
            <div class="extra-cell" v-if="data.festival && data.festival.lunar">
              <span class="ec-label">农历节</span>
              <span class="ec-val festival">{{ data.festival.lunar }}</span>
            </div>
            <div class="extra-cell" v-if="data.festival && data.festival.solar">
              <span class="ec-label">公历节</span>
              <span class="ec-val festival">{{ data.festival.solar }}</span>
            </div>
            <div class="extra-cell" v-if="data.festival && data.festival.legal">
              <span class="ec-label">假日</span>
              <span class="ec-val" :class="{ 'work-day': data.festival.isWork }">{{ data.festival.legal }}{{ data.festival.isWork ? '(班)' : '(休)' }}</span>
            </div>
            <div class="extra-cell" v-if="data.dogDay">
              <span class="ec-label">数九</span>
              <span class="ec-val">{{ data.dogDay }}</span>
            </div>
            <div class="extra-cell" v-if="data.plumRain">
              <span class="ec-label">梅雨</span>
              <span class="ec-val">{{ data.plumRain }}</span>
            </div>
            <div class="extra-cell" v-if="data.nineDay">
              <span class="ec-label">九星</span>
              <span class="ec-val">{{ data.nineDay }}</span>
            </div>
            <div class="extra-cell" v-if="data.hideHeavenStem">
              <span class="ec-label">藏干</span>
              <span class="ec-val">{{ data.hideHeavenStem }}</span>
            </div>
            <div class="extra-cell" v-if="data.yun">
              <span class="ec-label">运</span>
              <span class="ec-val">{{ data.yun.name }}</span>
            </div>
            <div class="extra-cell" v-if="data.yuan">
              <span class="ec-label">元</span>
              <span class="ec-val">{{ data.yuan.name }}</span>
            </div>
            <div class="extra-cell" v-if="data.xun">
              <span class="ec-label">旬</span>
              <span class="ec-val">{{ data.xun.name }}</span>
            </div>
          </div>
        </div>

        <!-- ─── MONTHLY CALENDAR ─── -->
        <div class="calendar-section">
          <div class="sec-header">
            <div class="sec-ornament"></div>
            <span class="cal-arrow left" @click="prevMonth()">◀</span>
            <span class="sec-title">{{ data.calendarMonth }}</span>
            <span class="cal-arrow right" @click="nextMonth()">▶</span>
            <div class="sec-line"></div>
          </div>
          <div class="cal-weekdays">
            <span v-for="w in ['日','一','二','三','四','五','六']" :key="w" class="cal-weekday">{{ w }}</span>
          </div>
          <div class="cal-days">
            <div v-for="(cd, idx) in data.calendarDays" :key="idx" class="cal-day" :class="{ empty: cd.empty, today: cd.isToday, 'has-festival': cd.festival || cd.legal }">
              <span class="cd-num">{{ cd.empty ? '' : cd.day }}</span>
              <span class="cd-lunar" v-if="!cd.empty">{{ cd.lunarShort }}</span>
              <span class="cd-festival" v-if="cd.festival">{{ cd.festival }}</span>
              <span class="cd-legal" v-if="cd.legal" :class="{ 'is-work': cd.isWork }">{{ cd.legal }}</span>
            </div>
          </div>
        </div>

        <!-- ─── CORNER MARKS ─── -->
        <div class="corner tl"></div>
        <div class="corner tr"></div>
        <div class="corner bl"></div>
        <div class="corner br"></div>

      </div>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@400;500;600;700;900&family=Noto+Sans+SC:wght@300;400;500;700&display=swap');

/* ─── ROOT: Dark glass theme ─── */
.cal {
  --red: #f43f5e;
  --red-dim: #be123c;
  --gold: #F5B041;
  --gold-bright: #FAD590;
  --gold-dim: #C9923A;
  --ink: #f8fafc;
  --text: #e2e8f0;
  --text-muted: #a5b4c4;
  --text-light: #6b7f94;
  --green: #34d399;
  --green-dim: #10b981;
  --purple: #c084fc;
  --surface: rgba(30, 41, 59, 0.6);
  --surface-light: rgba(30, 41, 59, 0.4);
  --surface-dark: rgba(15, 23, 42, 0.8);
  --border: rgba(255, 255, 255, 0.08);
  --border-light: rgba(255, 255, 255, 0.04);
  --border-dark: rgba(255, 255, 255, 0.12);
  --bg: #0f172a;
  --bg-card: rgba(30, 41, 59, 0.5);

  background: var(--bg);
  color: var(--text);
  font-family: 'Noto Serif SC', 'SimSun', serif;
  height: 100%;
  display: flex;
  align-items: stretch;
  justify-content: center;
  overflow-y: auto;
  padding: 8px;
}

/* ─── FRAME ─── */
.frame-outer {
  width: 100%;
  height: 100%;
  border: 1px solid var(--border-dark);
  border-radius: 12px;
  padding: 2px;
  background: var(--surface-dark);
  box-shadow:
    0 0 0 1px var(--border-light),
    0 4px 24px rgba(0,0,0,0.4),
    inset 0 1px 0 rgba(255,255,255,0.05);
  overflow: hidden;
}

.frame-inner {
  height: 100%;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  position: relative;
  background: linear-gradient(160deg, var(--surface) 0%, var(--surface-light) 100%);
}

/* ─── WEATHER ROW ─── */
.weather-row {
  display: grid;
  grid-template-columns: 1fr auto auto auto;
  align-items: center;
  gap: 14px;
  padding: 10px 14px;
  background: linear-gradient(135deg, rgba(56, 189, 248, 0.15), rgba(99, 102, 241, 0.1));
  border: 1px solid rgba(56, 189, 248, 0.25);
  border-radius: 10px;
}

.weather-main-block {
  display: flex;
  align-items: center;
  gap: 10px;
}

.weather-icon-lg {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.weather-icon-lg :deep(svg) {
  width: 100%;
  height: 100%;
}

.weather-primary {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.weather-temp {
  font-size: 26px;
  font-weight: 700;
  color: var(--ink);
  font-family: 'Noto Sans SC', sans-serif;
  line-height: 1.1;
}

.weather-feels {
  font-size: 10px;
  color: var(--text-muted);
  font-family: 'Noto Sans SC', sans-serif;
}

.weather-condition-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.weather-condition {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  font-family: 'Noto Sans SC', sans-serif;
}

.weather-stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px 10px;
}

.ws-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
}

.ws-label {
  font-size: 8px;
  color: var(--text-muted);
  letter-spacing: 1px;
}

.ws-val {
  font-size: 11px;
  font-weight: 600;
  color: var(--text);
  font-family: 'Noto Sans SC', sans-serif;
}


.weather-sun-times {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 10px;
  border-left: 1px solid var(--border);
  border-right: 1px solid var(--border);
}

.sun-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
}

.sun-label {
  font-size: 8px;
  color: var(--text-muted);
  letter-spacing: 1px;
}

.sun-val {
  font-size: 11px;
  font-weight: 600;
  color: #facc15;
  font-family: 'Noto Sans SC', sans-serif;
  text-shadow: 0 0 6px rgba(250,204,21,0.3);
}

.weather-tomorrow {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 10px;
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.weather-tomorrow:hover {
  background: rgba(99, 102, 241, 0.25);
}

.tomorrow-label {
  font-size: 8px;
  color: var(--text-muted);
  letter-spacing: 1px;
}

.tomorrow-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tomorrow-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.tomorrow-temp {
  font-size: 11px;
  font-weight: 600;
  color: var(--ink);
  font-family: 'Noto Sans SC', sans-serif;
}

/* ─── WEATHER POPUP ─── */
.weather-popup-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.weather-popup {
  background: linear-gradient(160deg, rgba(30, 41, 59, 0.95), rgba(15, 23, 42, 0.98));
  border: 1px solid rgba(56, 189, 248, 0.3);
  border-radius: 12px;
  padding: 16px;
  min-width: 280px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.popup-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--gold);
  letter-spacing: 2px;
}

.popup-close {
  font-size: 16px;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
}

.popup-close:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text);
}

.forecast-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.forecast-item {
  display: grid;
  grid-template-columns: 50px 40px 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: rgba(15, 23, 42, 0.5);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
}

.forecast-day {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.forecast-icon {
  width: 32px;
  height: 32px;
}

.forecast-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.forecast-condition {
  font-size: 12px;
  color: var(--text-muted);
}

.forecast-temp {
  font-size: 13px;
  font-weight: 600;
  color: var(--ink);
  white-space: nowrap;
}

/* ─── CORNER MARKS ─── */
.corner {
  position: absolute;
  width: 12px;
  height: 12px;
  border-color: var(--gold);
  border-style: solid;
  opacity: 0.6;
  box-shadow: 0 0 8px rgba(245,176,65,0.3);
}
.corner.tl { top: 6px; left: 6px; border-width: 2px 0 0 2px; }
.corner.tr { top: 6px; right: 6px; border-width: 2px 2px 0 0; }
.corner.bl { bottom: 6px; left: 6px; border-width: 0 0 2px 2px; }
.corner.br { bottom: 6px; right: 6px; border-width: 0 2px 2px 0; }

/* ─── DATE ROW ─── */
.date-row {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 6px;
  min-width: 0;
}

.date-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 12px;
  gap: 2px;
  flex: 1;
  min-width: 0;
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.date-cell.jieqi-cell .dc-main { color: var(--gold); }

.dc-label { font-size: 9px; color: var(--text-light); letter-spacing: 2px; text-transform: uppercase; }
.dc-main { font-size: 13px; font-weight: 700; color: var(--ink); display: flex; gap: 4px; align-items: baseline; flex-wrap: wrap; justify-content: center; white-space: nowrap; text-shadow: 0 0 8px rgba(248,250,252,0.15); }
.dc-main.gold { color: var(--gold); text-shadow: 0 0 10px rgba(245,176,65,0.3); }
.dc-sub { font-size: 9px; color: var(--text); letter-spacing: 1px; white-space: nowrap; font-weight: 600; }
.dc-time { font-size: 11px; color: var(--gold-bright); letter-spacing: 1px; white-space: nowrap; font-weight: 700; font-family: 'Noto Sans SC', monospace; text-shadow: 0 0 10px rgba(250,213,144,0.4); }
.dc-sub.wuhou { font-size: 8px; letter-spacing: 0; white-space: normal; text-align: center; }
.dc-sub.el-gold, .pb-el.el-gold { color: #e2e8f0; }
.dc-sub.el-wood, .pb-el.el-wood { color: #4ade80; }
.dc-sub.el-water, .pb-el.el-water { color: #38bdf8; }
.dc-sub.el-fire, .pb-el.el-fire { color: #fb7185; }
.dc-sub.el-earth, .pb-el.el-earth { color: #fbbf24; }

.l-year { color: var(--text-muted); font-weight: 400; font-size: 11px; display: block; width: 100%; text-align: center; letter-spacing: 1px; }
.l-month { color: var(--gold); font-weight: 700; text-shadow: 0 0 8px rgba(245,176,65,0.3); }
.l-month.leap { color: #fb923c; text-shadow: 0 0 8px rgba(251,146,60,0.3); }
.l-day { font-size: 15px; color: var(--gold-bright); text-shadow: 0 0 10px rgba(250,213,144,0.4); font-weight: 700; }

/* ─── PILLARS ROW ─── */
.pillars-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.pillar-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 8px 4px;
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.pillar-block.pillar-highlight {
  background: rgba(245,176,65,0.12);
  border-color: var(--gold);
  box-shadow: 0 0 16px rgba(245,176,65,0.2), inset 0 0 12px rgba(245,176,65,0.08);
}

.pb-header { font-size: 9px; color: var(--text-light); letter-spacing: 2px; }
.pb-name { font-size: 15px; font-weight: 700; color: var(--ink); letter-spacing: 2px; text-shadow: 0 0 8px rgba(248,250,252,0.1); }
.pillar-highlight .pb-name { color: var(--gold); text-shadow: 0 0 10px rgba(245,176,65,0.4); }
.pb-stems { font-size: 11px; color: var(--text-muted); letter-spacing: 1px; }
.pb-el { font-size: 9px; color: var(--text-light); }
.pb-nine { font-size: 9px; font-weight: 600; letter-spacing: 1px; padding-top: 2px; text-shadow: 0 0 6px rgba(245,176,65,0.2); }
.nine-white { color: #f8fafc; text-shadow: 0 0 8px rgba(248,250,252,0.5); }
.nine-black { color: #1e293b; }
.nine-blue { color: #60a5fa; text-shadow: 0 0 8px rgba(96,165,250,0.4); }
.nine-green { color: #4ade80; text-shadow: 0 0 8px rgba(74,222,128,0.4); }
.nine-yellow { color: #facc15; text-shadow: 0 0 8px rgba(250,204,21,0.4); }
.nine-red { color: #f87171; text-shadow: 0 0 8px rgba(248,113,113,0.4); }
.nine-purple { color: #c084fc; text-shadow: 0 0 8px rgba(192,132,252,0.4); }

/* ─── SECTION HEADERS ─── */
.sec-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.sec-header.no-ornament { margin-bottom: 6px; }

.sec-ornament {
  width: 8px;
  height: 8px;
  background: linear-gradient(135deg, var(--gold-bright), var(--gold));
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 8px rgba(245,176,65,0.5);
}

.sec-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--gold);
  letter-spacing: 3px;
  white-space: nowrap;
  text-shadow: 0 0 12px rgba(245,176,65,0.3);
}

.sec-line {
  flex: 1;
  height: 1px;
  background: linear-gradient(90deg, var(--border-dark), transparent);
}

.cur-hour-tag {
  font-size: 9px;
  color: var(--gold-dim);
  letter-spacing: 1px;
  margin-left: auto;
  font-weight: 600;
}

/* ─── STARS + INFO ROW ─── */
.stars-info-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

/* ─── MAIN GRID ─── */
.main-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.col-left, .col-right { display: flex; flex-direction: column; gap: 10px; }

.col-right { order: 2; }
.col-left { order: 1; }

/* ─── SECTION ─── */
.section {
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
}

/* ─── BAZI MATRIX ─── */
.bazi-matrix { display: flex; flex-direction: column; gap: 1px; margin-bottom: 8px; }

.bm-row { display: grid; grid-template-columns: 34px repeat(4, 1fr); gap: 1px; align-items: stretch; }
.bm-head { margin-bottom: 2px; }
.bm-head span { font-size: 9px; color: var(--text-muted); text-align: center; letter-spacing: 1px; }
.bm-head span:first-child { text-align: left; }
.bm-head span.gold { color: var(--gold); font-weight: 700; text-shadow: 0 0 8px rgba(245,176,65,0.3); }

.bm-lbl { font-size: 9px; color: var(--text-muted); display: flex; align-items: center; }

.bm-cell {
  font-size: 11px;
  background: rgba(15, 23, 42, 0.6);
  border-radius: 3px;
  padding: 5px 2px;
  text-align: center;
  font-weight: 600;
  letter-spacing: 1px;
  color: var(--text);
  display: flex;
  align-items: center;
  justify-content: center;
}

.bm-cell.na { color: var(--purple); font-size: 10px; text-shadow: 0 0 6px rgba(192,132,252,0.3); }
.bm-cell.el { color: var(--gold); }
.bm-cell.shishen { font-size: 10px; }
.bm-cell.pz { font-size: 9px; color: var(--text-muted); letter-spacing: 0; }
.bm-cell.zg { font-size: 10px; color: var(--purple); letter-spacing: 0; text-shadow: 0 0 6px rgba(192,132,252,0.3); }
.bm-cell.good { color: var(--green); text-shadow: 0 0 6px rgba(52,211,153,0.3); }
.bm-cell.bad { color: var(--red); text-shadow: 0 0 6px rgba(244,63,94,0.3); }
.bm-cell.el-gold.bm-cell { color: #e2e8f0; }
.bm-cell.el-wood.bm-cell { color: #4ade80; }
.bm-cell.el-water.bm-cell { color: #38bdf8; }
.bm-cell.el-fire.bm-cell { color: #fb7185; }
.bm-cell.el-earth.bm-cell { color: #fbbf24; }

/* Ming Table */
.ming-table { display: flex; gap: 3px; flex-wrap: wrap; padding-top: 8px; border-top: 1px dashed var(--border); }
.mt-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 5px 6px;
  background: rgba(15, 23, 42, 0.4);
  border-radius: 4px;
  border: 1px solid var(--border-light);
  min-width: 0;
  flex: 1;
}
.mt-item.mt-highlight { background: rgba(245,176,65,0.15); border-color: rgba(245,176,65,0.35); box-shadow: 0 0 10px rgba(245,176,65,0.1); }
.mt-lbl { font-size: 8px; color: var(--text-muted); }
.mt-val { font-size: 12px; color: var(--text); font-weight: 600; letter-spacing: 1px; }
.mt-val.gold { color: var(--gold); text-shadow: 0 0 8px rgba(245,176,65,0.3); }

/* ─── STARS ─── */
.stars-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 5px; }

.star-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 7px 4px;
  background: rgba(15, 23, 42, 0.4);
  border-radius: 6px;
  border: 1px solid var(--border-light);
}

.star-card.main-star {
  grid-column: span 3;
  flex-direction: row;
  gap: 10px;
  justify-content: center;
  background: rgba(245,176,65,0.1);
  border-color: rgba(245,176,65,0.25);
}

.star-card.gold-star {
  background: rgba(245,176,65,0.1);
  border-color: rgba(245,176,65,0.25);
}

.sc-key { font-size: 8px; color: var(--text-muted); letter-spacing: 1px; }
.sc-val { font-size: 13px; font-weight: 700; color: var(--text); letter-spacing: 1px; }
.sc-val.lg { font-size: 16px; text-shadow: 0 0 10px rgba(245,176,65,0.3); }
.sc-val.gold { color: var(--gold); text-shadow: 0 0 8px rgba(245,176,65,0.4); }
.sc-val.purple { color: var(--purple); text-shadow: 0 0 8px rgba(192,132,252,0.3); }
.sc-val.good { color: var(--green); }
.sc-val.bad { color: var(--red); }
.sc-sub { font-size: 8px; color: var(--text-light); }

/* ─── YI/JI ─── */
.yi-ji-split { display: grid; grid-template-columns: 1fr auto 1fr; gap: 8px; align-items: start; }
.yj-divider-v { width: 1px; align-self: stretch; background: linear-gradient(180deg, transparent, var(--border-dark), transparent); }
.yj-half { display: flex; flex-direction: column; gap: 5px; }
.yj-head { display: flex; align-items: center; gap: 5px; }
.yj-circle { width: 22px; height: 22px; border-radius: 50%; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.yj-circle svg { width: 12px; height: 12px; }
.yi-circle { background: var(--green); color: white; }
.ji-circle { background: var(--red); color: white; }
.yj-head span { font-size: 13px; font-weight: 700; color: var(--text); letter-spacing: 1px; }
.yj-chips { display: flex; flex-wrap: wrap; gap: 2px; }
.yj-chip { font-size: 9px; padding: 2px 6px; border-radius: 3px; letter-spacing: 0.5px; }
.yi-chip { background: rgba(52,211,153,0.15); color: var(--green); border: 1px solid rgba(52,211,153,0.25); text-shadow: 0 0 6px rgba(52,211,153,0.2); }
.ji-chip { background: rgba(244,63,94,0.15); color: var(--red); border: 1px solid rgba(244,63,94,0.2); text-shadow: 0 0 6px rgba(244,63,94,0.2); }

/* ─── GODS ─── */
.gods-2col { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
.god-col { display: flex; flex-direction: column; gap: 4px; }
.god-head { display: flex; align-items: center; gap: 5px; }
.god-dot { width: 6px; height: 6px; border-radius: 50%; }
.god-dot-good { background: var(--gold); box-shadow: 0 0 6px var(--gold); }
.god-dot-bad { background: var(--red); box-shadow: 0 0 6px var(--red); }
.god-head-txt { font-size: 10px; font-weight: 700; letter-spacing: 1px; }
.good-txt { color: var(--gold); }
.bad-txt { color: var(--red); }
.god-tags { display: flex; flex-wrap: wrap; gap: 2px; }
.god-tag { font-size: 9px; padding: 2px 6px; border-radius: 3px; }
.gt-good { background: rgba(245,176,65,0.15); color: var(--gold); border: 1px solid rgba(245,176,65,0.25); text-shadow: 0 0 6px rgba(245,176,65,0.2); }
.gt-bad { background: rgba(244,63,94,0.12); color: var(--red); border: 1px solid rgba(244,63,94,0.2); text-shadow: 0 0 6px rgba(244,63,94,0.2); }

/* ─── INFO GRID ─── */
.info-4grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; }
.info-chip-row { display: flex; flex-direction: column; gap: 1px; padding: 5px 7px; background: rgba(15, 23, 42, 0.4); border-radius: 4px; border: 1px solid var(--border-light); }
.info-chip-row.wide { grid-column: span 2; flex-direction: row; align-items: center; justify-content: space-between; }
.icr-label { font-size: 8px; color: var(--text-muted); letter-spacing: 1px; }
.icr-val { font-size: 11px; color: var(--text); font-weight: 600; letter-spacing: 1px; }
.icr-val.danger { color: var(--red); text-shadow: 0 0 8px rgba(244,63,94,0.3); }
.icr-val.purple { color: var(--purple); text-shadow: 0 0 8px rgba(192,132,252,0.3); }
.icr-val.sm { font-size: 9px; color: var(--text-muted); }
.icr-val.el-gold { color: #e2e8f0; }
.icr-val.el-wood { color: #4ade80; text-shadow: 0 0 6px rgba(74,222,128,0.3); }
.icr-val.el-water { color: #38bdf8; text-shadow: 0 0 6px rgba(56,189,248,0.3); }
.icr-val.el-fire { color: #fb7185; text-shadow: 0 0 6px rgba(251,113,133,0.3); }
.icr-val.el-earth { color: #fbbf24; text-shadow: 0 0 6px rgba(251,191,36,0.3); }
.bud-tags { display: flex; flex-wrap: wrap; gap: 2px; }
.bud-tag { font-size: 9px; padding: 1px 6px; background: rgba(249,115,22,0.15); color: #fb923c; border-radius: 3px; border: 1px solid rgba(249,115,22,0.2); text-shadow: 0 0 6px rgba(251,146,60,0.3); }

/* ─── BOTTOM ROW ─── */
.bottom-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 8px;
}

.bottom-row.four-col { grid-template-columns: 1fr 1fr 1fr 1fr; }
.bottom-row.three-col { grid-template-columns: 1fr 1fr 1fr; }

.bottom-section {
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px;
}

/* Seven Star */
.seven-star { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 4px; }
.seven-name { font-size: 15px; color: var(--gold); font-weight: 700; letter-spacing: 1px; text-shadow: 0 0 10px rgba(245,176,65,0.4); }
.seven-sub { font-size: 8px; color: var(--text-muted); }

/* Land */
.land-info { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 4px; }
.land-name { font-size: 14px; color: var(--text); font-weight: 700; letter-spacing: 1px; }
.land-sub { font-size: 8px; color: var(--text-muted); }

/* Nine Star */
.nine-star { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 4px; }
.nine-name { font-size: 15px; color: var(--purple); font-weight: 700; letter-spacing: 1px; text-shadow: 0 0 10px rgba(192,132,252,0.4); }
.nine-sub { font-size: 8px; color: var(--text-muted); }

/* TenStar & Terrain */
.tenstar-display, .terrain-display { display: flex; flex-wrap: wrap; justify-content: center; gap: 6px; padding: 4px; }
.tenstar-item, .terrain-item { display: flex; flex-direction: column; align-items: center; gap: 1px; }
.ts-label { font-size: 8px; color: var(--text-muted); }
.ts-val { font-size: 11px; color: var(--text); font-weight: 600; }

/* Duty */
.duty-single { display: flex; justify-content: center; align-items: center; padding: 4px; }
.duty-name { font-size: 17px; color: var(--gold); font-weight: 700; letter-spacing: 1px; text-shadow: 0 0 12px rgba(245,176,65,0.4); }

.hour-bar { display: grid; grid-template-columns: repeat(6, 1fr); gap: 3px; }
.hb-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 5px 2px;
  background: rgba(15, 23, 42, 0.4);
  border-radius: 4px;
  border: 1px solid transparent;
}
.hb-cell.cur {
  /* only text color changes */
}
.hb-name { font-size: 11px; color: var(--text-muted); }
.hb-dot { width: 5px; height: 5px; border-radius: 50%; background: var(--border-dark); }
.hb-cell.good .hb-dot { background: var(--green); box-shadow: 0 0 6px var(--green); }
.hb-cell.bad .hb-dot { background: var(--red); box-shadow: 0 0 6px var(--red); }
.hb-luck { font-size: 9px; font-weight: 700; }
.hb-cell.good .hb-luck { color: var(--green); text-shadow: 0 0 6px rgba(52,211,153,0.3); }
.hb-cell.bad .hb-luck { color: var(--red); text-shadow: 0 0 6px rgba(244,63,94,0.3); }
.hb-cell.cur .hb-name { color: var(--gold); font-weight: 700; }

/* Six Stars Strip */
.six-strip { display: flex; gap: 3px; flex-wrap: wrap; }
.six-chip { font-size: 9px; padding: 2px 6px; background: rgba(15, 23, 42, 0.4); border: 1px solid var(--border-light); border-radius: 4px; color: var(--text-muted); }
.six-chip.active { background: var(--gold); color: #0f172a; font-weight: 700; border-color: var(--gold); }
.six-single { display: flex; justify-content: center; align-items: center; padding: 4px; }
.six-name { font-size: 17px; color: var(--gold); font-weight: 700; letter-spacing: 1px; text-shadow: 0 0 12px rgba(245,176,65,0.4); }

/* Jianchu Strip */
.jianchu-strip { display: flex; gap: 2px; flex-wrap: wrap; }
.jc-chip { font-size: 9px; padding: 2px 5px; background: rgba(15, 23, 42, 0.4); border: 1px solid var(--border-light); border-radius: 4px; color: var(--text-muted); }
.jc-chip.active { background: var(--gold); color: #0f172a; font-weight: 700; border-color: var(--gold); }

/* ─── JIEQI FOOTER ─── */
.jieqi-footer {
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.jq-track { display: flex; flex-wrap: wrap; gap: 3px; padding: 4px 0; }
.jq-item { font-size: 9px; padding: 2px 7px; background: rgba(15, 23, 42, 0.4); border: 1px solid var(--border-light); border-radius: 4px; color: var(--text-muted); transition: all 0.2s; }
.jq-item.active { background: rgba(245,176,65,0.18); color: var(--gold); font-weight: 700; border-color: rgba(245,176,65,0.35); text-shadow: 0 0 8px rgba(245,176,65,0.3); }
.jq-item.past { opacity: 0.5; }

/* ─── EXTRA SECTION ─── */
.extra-section {
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.extra-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.extra-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 10px;
  background: rgba(15, 23, 42, 0.4);
  border: 1px solid var(--border-light);
  border-radius: 5px;
  min-width: 0;
}

.ec-label { font-size: 8px; color: var(--text-muted); letter-spacing: 1px; }
.ec-val { font-size: 12px; color: var(--text); font-weight: 600; letter-spacing: 1px; }
.ec-val.gold { color: var(--gold); text-shadow: 0 0 8px rgba(245,176,65,0.3); }
.ec-val.danger { color: var(--red); }
.ec-val.festival { color: #fb923c; text-shadow: 0 0 8px rgba(251,146,60,0.3); }
.ec-val.work-day { color: var(--green); text-shadow: 0 0 6px rgba(52,211,153,0.3); }

/* ─── CALENDAR SECTION ─── */
.calendar-section {
  background: var(--surface-dark);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px;
}

.cal-weekdays {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  margin-bottom: 6px;
}

.cal-weekday {
  text-align: center;
  font-size: 9px;
  color: var(--text-light);
  letter-spacing: 1px;
  padding: 4px 0;
  font-weight: 600;
}

.cal-arrow {
  cursor: pointer;
  font-size: 12px;
  color: var(--gold);
  padding: 2px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  user-select: none;
}

.cal-arrow:hover {
  background: rgba(245, 176, 65, 0.25);
  color: var(--gold-bright);
}

.cal-arrow.left { margin-right: 8px; }
.cal-arrow.right { margin-left: 8px; }

.cal-days {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}

.cal-day {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  padding: 5px 2px;
  background: rgba(15, 23, 42, 0.3);
  border: 1px solid var(--border-light);
  border-radius: 4px;
  min-height: 44px;
}

.cal-day.empty {
  background: transparent;
  border: none;
}

.cal-day.today {
  background: rgba(245, 176, 65, 0.18);
  border-color: var(--gold);
  box-shadow: 0 0 12px rgba(245,176,65,0.2);
}

.cal-day.has-festival {
  background: rgba(251, 146, 60, 0.1);
}

.cd-num {
  font-size: 12px;
  color: var(--text);
  font-weight: 700;
}

.cd-lunar {
  font-size: 8px;
  color: var(--text-muted);
}

.cd-festival {
  font-size: 8px;
  color: #fb923c;
  text-align: center;
  line-height: 1.1;
  text-shadow: 0 0 6px rgba(251,146,60,0.3);
}

.cd-legal {
  font-size: 8px;
  color: var(--green);
  text-shadow: 0 0 6px rgba(52,211,153,0.2);
}

.cd-legal.is-work {
  color: var(--red);
  text-shadow: 0 0 6px rgba(244,63,94,0.2);
}

/* ─── RESPONSIVE ─── */
@media (max-width: 480px) {
  .cal { padding: 4px; }
  .frame-outer { border-width: 1px; padding: 2px; }
  .frame-inner { padding: 10px 8px; gap: 8px; }
  .pillars-row { gap: 4px; }
  .pb-name { font-size: 13px; }
  .bottom-row { grid-template-columns: 1fr 1fr; }
  .jianchu-section { grid-column: span 2; }
  .bottom-row.four-col { grid-template-columns: 1fr 1fr; }
  .bottom-row.three-col { grid-template-columns: 1fr 1fr 1fr; }
  .date-row { flex-wrap: wrap; gap: 4px; }
  .date-cell { padding: 6px 8px; min-width: calc(50% - 2px); }
  .date-divider-v { display: none; }
  .l-day { font-size: 12px; }
  .yi-ji-split { grid-template-columns: 1fr; }
  .yj-divider-v { display: none; }
  .stars-row { grid-template-columns: repeat(2, 1fr); }
  .star-card.main-star { grid-column: span 2; }
  .gods-2col { grid-template-columns: 1fr; }
  .weather-row {
    grid-template-columns: 1fr 1fr;
    grid-template-rows: auto auto;
    gap: 10px;
  }
  .weather-main-block {
    grid-column: span 2;
  }
  .weather-stats-grid {
    grid-column: span 2;
    grid-template-columns: repeat(2, 1fr);
  }
  .weather-sun-times {
    flex-direction: row;
    border-left: none;
    border-right: none;
    border-top: 1px solid var(--border);
    border-bottom: 1px solid var(--border);
    padding: 6px 0;
    justify-content: space-around;
    width: 100%;
    grid-column: span 2;
  }
  .weather-tomorrow {
    grid-column: span 2;
  }
}
</style>

/** Deliberately stylized sky art; these values do not claim astronomical accuracy. */
export const celestialStyle = {
  sunNoonElevationDegrees: 55,
  sunRadiusDegrees: .7,
  sunHaloRadiusDegrees: 3,
  sunHaloOpacity: .16,
  moonRadiusDegrees: .8,
  moonlightIntensity: .65,
  moonlightPhaseExponent: 2,
  starCounts: { low: 384, medium: 768, high: 1536, ultra: 2048 },
} as const

/** Quiet hours fade in from 23:00 to 01:00, hold until 04:00, and end at 05:00.
 * A cyclic civil-time envelope keeps midnight and clock seeks continuous.
 */
export function deepNightAmount(localMinutes: number): number {
  if (!Number.isFinite(localMinutes)) throw new Error('Invalid atmosphere time')
  const hour = ((localMinutes / 60) % 24 + 24) % 24
  const smooth = (x: number) => x * x * (3 - 2 * x)
  if (hour >= 23) return smooth((hour - 23) / 2)
  if (hour < 1) return smooth((hour + 1) / 2)
  if (hour < 4) return 1
  if (hour < 5) return 1 - smooth(hour - 4)
  return 0
}

export function starVisibility(sunHeight: number, cloudCoverage: number, moonIllumination: number, moonHeight: number): number {
  const clamp = (value: number) => Math.max(0, Math.min(1, value))
  const twilight = clamp((.02 - sunHeight) / .22)
  const darkness = twilight * twilight * (3 - 2 * twilight)
  const moonlight = clamp(moonIllumination) * clamp(moonHeight * 4)
  return darkness * (1 - clamp(cloudCoverage)) ** 2 * (1 - .65 * moonlight)
}

export function stylizedSunDirection(localMinutes: number): [number, number, number] {
  if (!Number.isFinite(localMinutes)) throw new Error('Invalid civil time')
  const angle = (((localMinutes % 1440) + 1440) % 1440 - 360) / 1440 * 2 * Math.PI
  const tilt = celestialStyle.sunNoonElevationDegrees * Math.PI / 180
  return [Math.cos(angle), Math.sin(angle) * Math.sin(tilt), -Math.sin(angle) * Math.cos(tilt)]
}

/** One direction model for the visible bodies and the light they cast. Presets
 * retain their authored sun elevation; clock mode follows the civil-time orbit. */
export function celestialDirections(localMinutes: number, phase: { cycle: number; latitudeDegrees: number }, preset?: { elevationDegrees: number; setting: boolean }) {
  const angle = (preset?.elevationDegrees ?? 0) * Math.PI / 180
  const sun: [number, number, number] = preset ? [(preset.setting ? -1 : 1) * Math.cos(angle), Math.sin(angle), 0] : stylizedSunDirection(localMinutes)
  const tilt = celestialStyle.sunNoonElevationDegrees * Math.PI / 180
  const normal: [number, number, number] = preset ? [0, 0, 1] : [0, Math.cos(tilt), Math.sin(tilt)]
  const tangent = [normal[1] * sun[2] - normal[2] * sun[1], normal[2] * sun[0] - normal[0] * sun[2], normal[0] * sun[1] - normal[1] * sun[0]]
  const cycle = phase.cycle * Math.PI * 2, latitude = phase.latitudeDegrees * Math.PI / 180
  const moon = sun.map((value, i) => value * Math.cos(cycle) * Math.cos(latitude) - (tangent[i] ?? 0) * Math.sin(cycle) * Math.cos(latitude) + (normal[i] ?? 0) * Math.sin(latitude)) as [number, number, number]
  return { sun, moon }
}

/** Stylized reflected moonlight: a quarter moon is substantially dimmer than
 * a full moon. Below-horizon bodies contribute no direct light. Ambient sky
 * fill remains a separate navigation aid, including on moonless nights. */
export function celestialKey(directions: ReturnType<typeof celestialDirections>, illumination: number, cloudCoverage: number, daylightIntensity: number) {
  const clamp = (x: number) => Math.max(0, Math.min(1, x))
  const smooth = (x: number) => { const t = clamp(x); return t * t * (3 - 2 * t) }
  const day = smooth(directions.sun[1] / .12)
  const sun = Math.max(0, daylightIntensity) * day
  const moon = celestialStyle.moonlightIntensity * clamp(illumination) ** celestialStyle.moonlightPhaseExponent * smooth(directions.moon[1] / .2) * (1 - clamp(cloudCoverage)) ** 2 * (1 - day)
  const direction = directions.sun.map((value, i) => value * sun + (directions.moon[i] ?? 0) * moon) as [number, number, number]
  const length = Math.hypot(...direction)
  return { intensity: sun + moon, moonIntensity: moon, direction: length > 1e-9 ? direction.map(value => value / length) as [number, number, number] : [0, 1, 0] as [number, number, number] }
}

const radians = Math.PI / 180
const wrapDegrees = (angle: number) => ((angle % 360) + 360) % 360
const sin = (angle: number) => Math.sin(wrapDegrees(angle) * radians)
const cos = (angle: number) => Math.cos(wrapDegrees(angle) * radians)

function trueAnomaly(meanDegrees: number, eccentricity: number): number {
  const mean = wrapDegrees(meanDegrees) * radians
  let eccentric = mean
  for (let iteration = 0; iteration < 8; iteration++) {
    eccentric -= (eccentric - eccentricity * Math.sin(eccentric) - mean) / (1 - eccentricity * Math.cos(eccentric))
  }
  return Math.atan2(Math.sqrt(1 - eccentricity * eccentricity) * Math.sin(eccentric), Math.cos(eccentric) - eccentricity) / radians
}

/** Geocentric lunar phase from orbital elements and dominant perturbations.
 * Mathematical reference: Paul Schlyter, sections 3–9 and 15:
 * https://stjarnhimlen.se/comp/ppcomp.html
 * No observer location, parallax, libration, nutation or eclipse model is implied.
 */
export function lunarPhase(utcMilliseconds: number) {
  if (!Number.isFinite(utcMilliseconds) || Math.abs(utcMilliseconds) > 1e15) throw new Error('Invalid lunar UTC instant')
  const days = (utcMilliseconds - Date.UTC(1999, 11, 31)) / 86400000
  const solarPerihelion = 282.9404 + .0000470935 * days
  const solarMean = 356.047 + .9856002585 * days
  const solarEccentricity = .016709 - .000000001151 * days
  const sunLongitudeDegrees = wrapDegrees(solarPerihelion + trueAnomaly(solarMean, solarEccentricity))
  const node = 125.1228 - .0529538083 * days
  const perihelion = 318.0634 + .1643573223 * days
  const mean = 115.3654 + 13.0649929509 * days
  const orbit = trueAnomaly(mean, .0549) + perihelion
  const x = cos(node) * cos(orbit) - sin(node) * sin(orbit) * cos(5.1454)
  const y = sin(node) * cos(orbit) + cos(node) * sin(orbit) * cos(5.1454)
  const z = sin(orbit) * sin(5.1454)
  const elongation = mean + perihelion + node - solarMean - solarPerihelion
  const latitudeArgument = mean + perihelion
  const longitudeCorrection = -1.274 * sin(mean - 2 * elongation) + .658 * sin(2 * elongation) - .186 * sin(solarMean)
    - .059 * sin(2 * mean - 2 * elongation) - .057 * sin(mean - 2 * elongation + solarMean) + .053 * sin(mean + 2 * elongation)
    + .046 * sin(2 * elongation - solarMean) + .041 * sin(mean - solarMean) - .035 * sin(elongation) - .031 * sin(mean + solarMean)
    - .015 * sin(2 * latitudeArgument - 2 * elongation) + .011 * sin(mean - 4 * elongation)
  const latitudeCorrection = -.173 * sin(latitudeArgument - 2 * elongation) - .055 * sin(mean - latitudeArgument - 2 * elongation)
    - .046 * sin(mean + latitudeArgument - 2 * elongation) + .033 * sin(latitudeArgument + 2 * elongation) + .017 * sin(2 * mean + latitudeArgument)
  const longitudeDegrees = wrapDegrees(Math.atan2(y, x) / radians + longitudeCorrection)
  const latitudeDegrees = Math.atan2(z, Math.hypot(x, y)) / radians + latitudeCorrection
  const cycle = wrapDegrees(longitudeDegrees - sunLongitudeDegrees) / 360
  const illumination = Math.max(0, Math.min(1, (1 - cos(longitudeDegrees - sunLongitudeDegrees) * cos(latitudeDegrees)) / 2))
  return { cycle, illumination, waxing: cycle < .5, longitudeDegrees, latitudeDegrees, sunLongitudeDegrees }
}

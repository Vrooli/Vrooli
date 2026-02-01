/**
 * Sun/Moon position calculations for continuous time-based sky rendering.
 *
 * All calculations work with continuous time (0-24 hours).
 * Sun arcs from East (6h) → overhead South (12h) → West (18h)
 * Moon is positioned opposite to the sun.
 */

/**
 * Calculate sun position based on hour (0-24).
 * Sun rises at ~6am (East), peaks at noon (South/overhead), sets at ~18pm (West).
 *
 * @param hour - Continuous hour value (0-24, e.g., 14.5 = 2:30 PM)
 * @returns [x, y, z] position for the sun
 */
export function calculateSunPosition(hour: number): [number, number, number] {
  // Normalize hour to 0-24 range
  const normalizedHour = ((hour % 24) + 24) % 24

  // Convert hour to angle (0h = midnight, 12h = noon)
  // Sun rises at 6h (90°), peaks at 12h (0°), sets at 18h (-90°)
  // Full cycle: 24h = 360° = 2π radians
  const hourAngle = ((normalizedHour - 12) / 24) * Math.PI * 2

  // Radius of sun orbit
  const orbitRadius = 40

  // Calculate position
  // x: East-West (negative = East at sunrise, positive = West at sunset)
  // y: Height above horizon (peaks at noon)
  // z: North-South offset (sun arcs through southern sky in northern hemisphere)
  // Note: Using -sin for x so sunrise (6h) is positive x (East) and sunset (18h) is negative x (West)
  const x = -Math.sin(hourAngle) * orbitRadius
  const y = Math.cos(hourAngle) * orbitRadius * 0.8 // Scale down for more realistic arc
  const z = Math.cos(hourAngle) * orbitRadius * 0.3 // Slight south offset

  return [x, y, z]
}

/**
 * Calculate moon position based on hour (0-24).
 * Moon is positioned roughly opposite to the sun.
 *
 * @param hour - Continuous hour value (0-24)
 * @returns [x, y, z] position for the moon
 */
export function calculateMoonPosition(hour: number): [number, number, number] {
  // Moon is 12 hours offset from sun (opposite side)
  const sunPos = calculateSunPosition(hour + 12)
  // Slightly different orbit radius for visual variety
  return [sunPos[0] * 0.9, sunPos[1] * 0.95, sunPos[2] * 0.9]
}

/**
 * Calculate star visibility opacity based on hour.
 * Stars fade in at dusk (18-20h) and fade out at dawn (4-6h).
 *
 * @param hour - Continuous hour value (0-24)
 * @returns Opacity value (0-1) for stars
 */
export function calculateStarOpacity(hour: number): number {
  // Normalize hour
  const h = ((hour % 24) + 24) % 24

  // Fully dark (stars visible): 20h-4h
  // Dawn transition: 4h-6h (fade out)
  // Fully light (no stars): 6h-18h
  // Dusk transition: 18h-20h (fade in)

  if (h >= 20 || h < 4) {
    // Night time - full opacity
    return 1
  } else if (h >= 4 && h < 6) {
    // Dawn - fade out
    return 1 - (h - 4) / 2
  } else if (h >= 6 && h < 18) {
    // Day time - no stars
    return 0
  } else {
    // Dusk (18-20) - fade in
    return (h - 18) / 2
  }
}

/**
 * Check if it's currently night time (sun below horizon).
 *
 * @param hour - Continuous hour value (0-24)
 * @returns true if sun is below horizon
 */
export function isNightTime(hour: number): boolean {
  const h = ((hour % 24) + 24) % 24
  return h < 6 || h >= 18
}

/**
 * Sky color palette for different times of day.
 */
export interface SkyColors {
  top: string
  middle: string
  bottom: string
}

/**
 * Calculate sky gradient colors based on continuous time.
 * Used for the custom gradient shader fallback (abstract-space scene).
 *
 * @param hour - Continuous hour value (0-24)
 * @returns Sky gradient colors for top, middle, and bottom of dome
 */
export function calculateSkyColors(hour: number): SkyColors {
  const h = ((hour % 24) + 24) % 24

  // Define color keyframes at specific times
  const colorKeyframes: { hour: number; colors: SkyColors }[] = [
    { hour: 0, colors: { top: '#0a0a1a', middle: '#1a1a3a', bottom: '#0f0f2f' } }, // Midnight
    { hour: 4, colors: { top: '#0a1020', middle: '#1a2040', bottom: '#1a1a3a' } }, // Pre-dawn
    { hour: 6, colors: { top: '#2d4a6e', middle: '#87CEEB', bottom: '#FFE4B5' } }, // Sunrise
    { hour: 8, colors: { top: '#5a8fc9', middle: '#87CEEB', bottom: '#ADD8E6' } }, // Morning
    { hour: 12, colors: { top: '#4a90d9', middle: '#87CEEB', bottom: '#ADD8E6' } }, // Noon
    { hour: 16, colors: { top: '#6a9fd9', middle: '#87CEEB', bottom: '#FFE4B5' } }, // Afternoon
    { hour: 18, colors: { top: '#2C1810', middle: '#FF6B35', bottom: '#F7C59F' } }, // Sunset
    { hour: 20, colors: { top: '#0f1020', middle: '#2a2050', bottom: '#1a1040' } }, // Dusk
    { hour: 24, colors: { top: '#0a0a1a', middle: '#1a1a3a', bottom: '#0f0f2f' } }, // Midnight (wrap)
  ]

  // Find surrounding keyframes
  const lastKeyframe = colorKeyframes[colorKeyframes.length - 2]
  const firstKeyframe = colorKeyframes[0]
  if (!lastKeyframe || !firstKeyframe) {
    return { top: '#0a0a1a', middle: '#1a1a3a', bottom: '#0f0f2f' }
  }
  let prev = lastKeyframe // Second to last (23:59)
  let next = firstKeyframe // First (00:00)

  for (let i = 0; i < colorKeyframes.length - 1; i++) {
    const current = colorKeyframes[i]
    const following = colorKeyframes[i + 1]
    if (current && following && h >= current.hour && h < following.hour) {
      prev = current
      next = following
      break
    }
  }

  // Interpolate between keyframes
  const range = next.hour - prev.hour || 24
  const t = (h - prev.hour) / range

  return {
    top: interpolateColor(prev.colors.top, next.colors.top, t),
    middle: interpolateColor(prev.colors.middle, next.colors.middle, t),
    bottom: interpolateColor(prev.colors.bottom, next.colors.bottom, t),
  }
}

/**
 * Interpolate between two hex colors.
 */
function interpolateColor(color1: string, color2: string, t: number): string {
  const r1 = parseInt(color1.slice(1, 3), 16)
  const g1 = parseInt(color1.slice(3, 5), 16)
  const b1 = parseInt(color1.slice(5, 7), 16)

  const r2 = parseInt(color2.slice(1, 3), 16)
  const g2 = parseInt(color2.slice(3, 5), 16)
  const b2 = parseInt(color2.slice(5, 7), 16)

  const r = Math.round(r1 + (r2 - r1) * t)
  const g = Math.round(g1 + (g2 - g1) * t)
  const b = Math.round(b1 + (b2 - b1) * t)

  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`
}

/**
 * Calculate lighting parameters based on time.
 * Used to sync directional light with sun position.
 */
export interface LightingParams {
  /** Direction light should point (normalized sun position) */
  direction: [number, number, number]
  /** Light color based on time of day */
  color: string
  /** Light intensity (brighter at noon, dimmer at twilight) */
  intensity: number
  /** Ambient light color */
  ambientColor: string
  /** Ambient light intensity */
  ambientIntensity: number
}

/**
 * Calculate lighting parameters based on continuous time.
 *
 * @param hour - Continuous hour value (0-24)
 * @returns Lighting configuration for the time of day
 */
export function calculateLighting(hour: number): LightingParams {
  const sunPos = calculateSunPosition(hour)
  const h = ((hour % 24) + 24) % 24

  // Normalize sun position for direction
  const length = Math.sqrt(sunPos[0] ** 2 + sunPos[1] ** 2 + sunPos[2] ** 2)
  const direction: [number, number, number] = [
    sunPos[0] / length,
    Math.max(0.1, sunPos[1] / length), // Keep some height even at night
    sunPos[2] / length,
  ]

  // Light color and intensity based on time
  let color: string
  let intensity: number
  let ambientColor: string
  let ambientIntensity: number

  if (h >= 6 && h < 8) {
    // Sunrise
    const t = (h - 6) / 2
    color = interpolateColor('#FFB366', '#FFFFFF', t)
    intensity = 0.8 + t * 0.7
    ambientColor = interpolateColor('#FFE4B5', '#FFFFFF', t)
    ambientIntensity = 0.3 + t * 0.3
  } else if (h >= 8 && h < 16) {
    // Day
    color = '#FFFFFF'
    intensity = 1.5
    ambientColor = '#FFFFFF'
    ambientIntensity = 0.6
  } else if (h >= 16 && h < 18) {
    // Afternoon to sunset
    const t = (h - 16) / 2
    color = interpolateColor('#FFFFFF', '#FF9966', t)
    intensity = 1.5 - t * 0.5
    ambientColor = interpolateColor('#FFFFFF', '#FFDDCC', t)
    ambientIntensity = 0.6 - t * 0.2
  } else if (h >= 18 && h < 20) {
    // Sunset to dusk
    const t = (h - 18) / 2
    color = interpolateColor('#FF9966', '#6688BB', t)
    intensity = 1.0 - t * 0.4
    ambientColor = interpolateColor('#FFDDCC', '#334466', t)
    ambientIntensity = 0.4 - t * 0.1
  } else {
    // Night
    color = '#6688BB'
    intensity = 0.6
    ambientColor = '#334466'
    ambientIntensity = 0.3
  }

  return {
    direction,
    color,
    intensity,
    ambientColor,
    ambientIntensity,
  }
}

/**
 * Convert lighting params to a LightingPreset.
 * This creates a complete preset with ambient and directional lights
 * configured based on the time-derived lighting params.
 */
export interface LightingPreset {
  ambient: { color: string; intensity: number }
  directional: { position: [number, number, number]; color: string; intensity: number; castShadow?: boolean; shadowMapSize?: number }[]
  point?: { position: [number, number, number]; color: string; intensity: number; distance?: number; decay?: number }[]
}

/**
 * Calculate a complete LightingPreset from continuous time.
 * This is the main function to use for environment lighting.
 *
 * @param hour - Continuous hour value (0-24)
 * @returns Complete lighting preset for the time of day
 */
export function calculateLightingPreset(hour: number): LightingPreset {
  const params = calculateLighting(hour)

  // Scale direction to a reasonable position (40 units from origin)
  const scale = 40
  const position: [number, number, number] = [
    params.direction[0] * scale,
    Math.max(5, params.direction[1] * scale), // Keep light above ground
    params.direction[2] * scale,
  ]

  return {
    ambient: {
      color: params.ambientColor,
      intensity: params.ambientIntensity,
    },
    directional: [
      {
        position,
        color: params.color,
        intensity: params.intensity,
        castShadow: true,
        shadowMapSize: 2048,
      },
    ],
  }
}

/**
 * Format hour value as human-readable time string.
 *
 * @param hour - Continuous hour value (0-24)
 * @returns Formatted time string (e.g., "2:30 PM")
 */
export function formatTimeFromHour(hour: number): string {
  const h = ((hour % 24) + 24) % 24
  const hours = Math.floor(h)
  const minutes = Math.round((h - hours) * 60)

  const period = hours >= 12 ? 'PM' : 'AM'
  const displayHours = hours === 0 ? 12 : hours > 12 ? hours - 12 : hours
  const displayMinutes = minutes.toString().padStart(2, '0')

  return `${displayHours}:${displayMinutes} ${period}`
}

/**
 * Get current system time as hour value.
 *
 * @returns Current time as continuous hour (0-24)
 */
export function getCurrentTimeAsHour(): number {
  const now = new Date()
  return now.getHours() + now.getMinutes() / 60
}

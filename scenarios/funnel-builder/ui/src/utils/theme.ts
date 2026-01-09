import type { FunnelTheme } from '../types'

export const DEFAULT_PRIMARY_COLOR = '#0ea5e9'
export const DEFAULT_BORDER_RADIUS = '0.5rem'

const SYSTEM_FONT_STACK = "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
const DEFAULT_FONT_STACK = `'Inter', 'Inter var', ${SYSTEM_FONT_STACK}`

const KNOWN_FONT_STACKS: Record<string, string> = {
  Inter: DEFAULT_FONT_STACK,
  Poppins: `'Poppins', 'Helvetica Neue', Helvetica, Arial, sans-serif`,
  Rubik: `'Rubik', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif`,
  'DM Sans': `'DM Sans', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif`,
  'IBM Plex Sans': `'IBM Plex Sans', 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif`
}

const clamp = (value: number, min: number, max: number): number => {
  if (Number.isNaN(value)) {
    return min
  }
  return Math.min(Math.max(value, min), max)
}

const toFullHex = (hex: string): string => {
  if (hex.length === 3) {
    return hex
      .split('')
      .map((char) => char + char)
      .join('')
  }
  return hex
}

const hexToRgb = (hex: string): [number, number, number] => {
  const sanitized = sanitizeHexColor(hex)
  const value = sanitized.slice(1)
  const numeric = parseInt(value, 16)
  const r = (numeric >> 16) & 0xff
  const g = (numeric >> 8) & 0xff
  const b = numeric & 0xff
  return [r, g, b]
}

const rgbChannelToHex = (channel: number): string => {
  const clamped = clamp(Math.round(channel), 0, 255)
  const hex = clamped.toString(16)
  return hex.length === 1 ? `0${hex}` : hex
}

const rgbToHex = (r: number, g: number, b: number): string => `#${rgbChannelToHex(r)}${rgbChannelToHex(g)}${rgbChannelToHex(b)}`

const mixChannel = (channel: number, factor: number, target: number): number => channel + (target - channel) * factor

export const sanitizeHexColor = (value?: string | null, fallback = DEFAULT_PRIMARY_COLOR): string => {
  if (typeof value !== 'string') {
    return fallback
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return fallback
  }

  const match = trimmed.match(/^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/)
  if (!match) {
    return fallback
  }

  const fullHex = toFullHex(match[1].toLowerCase())
  return `#${fullHex}`
}

export const lightenColor = (color: string, amount: number): string => {
  const factor = clamp(amount, 0, 1)
  const [r, g, b] = hexToRgb(color)
  return rgbToHex(
    mixChannel(r, factor, 255),
    mixChannel(g, factor, 255),
    mixChannel(b, factor, 255)
  )
}

export const darkenColor = (color: string, amount: number): string => {
  const factor = clamp(amount, 0, 1)
  const [r, g, b] = hexToRgb(color)
  return rgbToHex(
    mixChannel(r, factor, 0),
    mixChannel(g, factor, 0),
    mixChannel(b, factor, 0)
  )
}

const escapeFontValue = (value: string): string => value.replace(/'/g, "\\'")

export const getFontStack = (value?: string | null): string => {
  if (!value) {
    return DEFAULT_FONT_STACK
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return DEFAULT_FONT_STACK
  }

  const known = KNOWN_FONT_STACKS[trimmed]
  if (known) {
    return known
  }

  return `'${escapeFontValue(trimmed)}', ${SYSTEM_FONT_STACK}`
}

export interface ThemeTokens {
  primary: string
  primaryHover: string
  primaryPressed: string
  primarySoft: string
  primarySoftAlt: string
  primaryBorder: string
  badgeBackground: string
  badgeText: string
  ring: string
  borderRadius: string
  fontFamily: string
  backgroundGradient: string
}

export const buildThemeTokens = (theme: FunnelTheme | null | undefined): ThemeTokens => {
  const primaryColor = sanitizeHexColor(theme?.primaryColor)
  const borderRadius = (theme?.borderRadius && theme.borderRadius.trim()) || DEFAULT_BORDER_RADIUS
  const fontFamily = getFontStack(theme?.fontFamily)

  const primaryHover = lightenColor(primaryColor, 0.12)
  const primaryPressed = darkenColor(primaryColor, 0.18)
  const primarySoft = lightenColor(primaryColor, 0.76)
  const primarySoftAlt = lightenColor(primaryColor, 0.62)
  const primaryBorder = darkenColor(primaryColor, 0.32)
  const badgeBackground = lightenColor(primaryColor, 0.68)
  const badgeText = darkenColor(primaryColor, 0.45)
  const ring = lightenColor(primaryColor, 0.24)
  const gradientStart = lightenColor(primaryColor, 0.88)
  const gradientEnd = lightenColor(primaryColor, 0.7)

  return {
    primary: primaryColor,
    primaryHover,
    primaryPressed,
    primarySoft,
    primarySoftAlt,
    primaryBorder,
    badgeBackground,
    badgeText,
    ring,
    borderRadius,
    fontFamily,
    backgroundGradient: `linear-gradient(135deg, ${gradientStart} 0%, ${gradientEnd} 100%)`
  }
}


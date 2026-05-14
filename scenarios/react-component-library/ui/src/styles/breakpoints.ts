export const APP_BREAKPOINTS = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  "2xl": 1536,
} as const;

export const APP_SCREENS = {
  sm: `${APP_BREAKPOINTS.sm}px`,
  md: `${APP_BREAKPOINTS.md}px`,
  lg: `${APP_BREAKPOINTS.lg}px`,
  xl: `${APP_BREAKPOINTS.xl}px`,
  "2xl": `${APP_BREAKPOINTS["2xl"]}px`,
} as const;

export const APP_MEDIA_QUERIES = {
  mobile: `(max-width: ${APP_BREAKPOINTS.md - 1}px)`,
  tablet: `(min-width: ${APP_BREAKPOINTS.md}px) and (max-width: ${APP_BREAKPOINTS.lg - 1}px)`,
  desktop: `(min-width: ${APP_BREAKPOINTS.lg}px)`,
} as const;

/** Shared filename convention. Omitted period preserves legacy day-only names. */
export interface GoldenCase {
  scene: string
  profile: string
  period?: string | null
  weather?: string | null
}

export function goldenKey({ scene, profile, period, weather }: GoldenCase): string {
  return `${scene}-${profile}${period ? `-${period}` : ''}${weather ? `-weather-${weather}` : ''}`
}

/** Omitted selectors mean day/clear, so their goldens share canonical pixels. */
export function goldenAliases(input: GoldenCase): string[] {
  const period = input.period ?? 'day'
  const weather = input.weather ?? 'clear'
  return (period === 'day' ? [period, undefined] : [period]).flatMap((p) =>
    (weather === 'clear' ? [weather, undefined] : [weather]).map((w) => goldenKey({ ...input, period: p, weather: w })),
  )
}

export interface CaptureConfig {
  budgets: { scenes: Record<string, unknown> }
  quality: { profiles: Record<string, unknown> }
  lighting: { periods: Record<string, unknown> }
  weather: { states: Record<string, unknown> }
}

/** Read authoritative keys, including future keys unknown to today's UI types. */
export function captureAxes(config: CaptureConfig) {
  return {
    scenes: Object.keys(config.budgets.scenes),
    profiles: Object.keys(config.quality.profiles),
    periods: Object.keys(config.lighting.periods),
    weather: Object.keys(config.weather.states),
  }
}

export function captureMatrix(config: CaptureConfig): GoldenCase[] {
  const axes = captureAxes(config)
  return axes.scenes.flatMap((scene) => axes.profiles.flatMap((profile) =>
    axes.periods.flatMap((period) => axes.weather.map((weather) => ({ scene, profile, period, weather }))),
  ))
}

/** Every supported capture spelling, including legacy day/clear aliases. */
export function expectedGoldenKeys(config: CaptureConfig): string[] {
  const axes = captureAxes(config)
  return axes.scenes.flatMap((scene) => axes.profiles.flatMap((profile) =>
    [undefined, ...axes.periods].flatMap((period) => [undefined, ...axes.weather].map((weather) => goldenKey({ scene, profile, period, weather }))),
  )).sort()
}

export function goldenCoverage(expected: readonly string[], files: readonly string[]) {
  const wanted = new Set(expected.map((key) => `${key}.png`))
  const actual = new Set(files.filter((name) => name.endsWith('.png')))
  return {
    missing: [...wanted].filter((name) => !actual.has(name)).sort(),
    orphaned: [...actual].filter((name) => !wanted.has(name)).sort(),
  }
}

import pixelmatch from 'pixelmatch'

/** Weather changes can be chromatic without the large luminance delta of night.
 * Distinct-state sensitivity is separate from same-state golden tolerance. */
export function weatherContrast(a, b, budgets) {
  if (a.width !== b.width || a.height !== b.height) throw new Error('Contrast images have different dimensions')
  const pixels = pixelmatch(a.data, b.data, null, a.width, a.height, { threshold: budgets.weatherPixelTolerance })
  const ratio = pixels / (a.width * a.height)
  return { pixels, ratio, pass: ratio >= budgets.periodPixelDelta }
}

/** A failed capture is failed evidence, never a contrasting error-page image.
 * Keep reporting other pairs even when one image is absent or malformed. */
export function captureContrasts(records, { weatherStates = [], periods = [], budgets, readImage }) {
  const results = []
  const compare = (left, right, name, kind) => {
    let ratio = null
    let pass = false
    let detail
    try {
      if (!left || !right) throw new Error('capture pair is incomplete')
      for (const record of [left, right]) {
        if (!record.diagnostics || !record.checks?.some(check => check.id === 'snapshot' && check.pass)) {
          throw new Error(`capture ${record.name} has no valid world snapshot`)
        }
      }
      const a = readImage(left.name)
      const b = readImage(right.name)
      if (a.width !== b.width || a.height !== b.height) throw new Error('contrast images have different dimensions')
      const pixels = pixelmatch(a.data, b.data, null, a.width, a.height, { threshold: kind === 'weather' ? budgets.weatherPixelTolerance : 0.1 })
      ratio = pixels / (a.width * a.height)
      pass = ratio >= budgets.periodPixelDelta
      detail = `${(ratio * 100).toFixed(2)}% pixels differ (min ${(budgets.periodPixelDelta * 100).toFixed(0)}%)`
    } catch (error) {
      detail = `contrast unavailable: ${error instanceof Error ? error.message : String(error)}`
    }
    results.push({ name, pass, ratio, minimum: budgets.periodPixelDelta, pixelTolerance: kind === 'weather' ? budgets.weatherPixelTolerance : 0.1, checks: [{ id: `${kind}-delta`, pass, detail }], diagnostics: null, sim: null })
  }
  const groupBy = (axis) => {
    const groups = new Map()
    for (const record of records.filter(record => record.scene)) {
      const key = JSON.stringify([record.scene, record.profile, axis === 'weather' ? record.period : record.weather, record.seed, record.actors])
      if (!groups.has(key)) groups.set(key, new Map())
      groups.get(key).set(record[axis], record)
    }
    return groups.values()
  }
  for (const group of groupBy('weather')) {
    const sample = group.values().next().value
    for (let left = 0; left < weatherStates.length; left += 1) for (let right = left + 1; right < weatherStates.length; right += 1) {
      const a = group.get(weatherStates[left])
      const b = group.get(weatherStates[right])
      const aName = a?.name ?? `${sample.name}-missing-${weatherStates[left]}`
      compare(a, b, `${aName}-vs-${weatherStates[right]}`, 'weather')
    }
  }
  if (periods.includes('day') && periods.includes('night')) {
    for (const group of groupBy('period')) {
      const day = group.get('day')
      const sample = day ?? group.values().next().value
      compare(day, group.get('night'), `${sample.name.replace('-day', '')}-period-delta`, 'period')
    }
  }
  return results
}

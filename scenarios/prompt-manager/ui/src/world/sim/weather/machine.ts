import type { PeriodId, WeatherId, WeatherTuning } from '../../config'
import type { Rng } from '../rng'

export interface WeatherState {
  state: WeatherId
  since: number
  until: number
  pressure: number
}

function duration(id: WeatherId, rng: Rng, tuning: WeatherTuning): number {
  const preset = tuning.states[id]
  return rng.range(preset.minSeconds, preset.maxSeconds)
}

export function initialWeather(now: number, rng: Rng, tuning: WeatherTuning): WeatherState {
  return { state: 'clear', since: now, until: now + duration('clear', rng, tuning), pressure: 0 }
}

export function stepWeather(current: WeatherState, now: number, pressure: number, rng: Rng, tuning: WeatherTuning, period: PeriodId): WeatherState {
  if (now < current.until) return { ...current, pressure }
  let next: WeatherId
  if (current.state === 'clear') next = rng.next() < pressure ? 'cloudy' : 'clear'
  else if (current.state === 'cloudy') {
    if (rng.next() < pressure) next = period === 'night' || period === 'dawn' ? (rng.next() < pressure ? 'snow' : 'rain') : 'rain'
    else next = 'clear'
  } else next = rng.next() < pressure ? 'cloudy' : 'clear'
  return { state: next, since: now, until: now + duration(next, rng, tuning), pressure }
}

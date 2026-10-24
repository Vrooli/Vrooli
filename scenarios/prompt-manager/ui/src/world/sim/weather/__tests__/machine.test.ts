import { describe, expect, it } from 'vitest'
import { tuning } from '../../../config'
import { Rng } from '../../rng'
import { initialWeather, stepWeather } from '..'

describe('weather machine', () => {
  it('is deterministic and respects configured duration bounds', () => {
    const run = () => {
      const rng = new Rng(7)
      let state = initialWeather(0, rng, tuning.weather)
      const sequence = [state.state]
      for (let now = 300; now <= 1800; now += 300) {
        state = stepWeather(state, now, 0.8, rng, tuning.weather, 'night')
        sequence.push(state.state)
        expect(state.until - state.since).toBeGreaterThanOrEqual(tuning.weather.states[state.state].minSeconds)
        expect(state.until - state.since).toBeLessThanOrEqual(tuning.weather.states[state.state].maxSeconds)
      }
      return sequence
    }
    expect(run()).toEqual(run())
  })

  it('moves clear through cloudy toward precipitation under pressure', () => {
    const rng = new Rng(1)
    const clear = { state: 'clear' as const, since: 0, until: 0, pressure: 0 }
    const cloudy = stepWeather(clear, 1, 1, rng, tuning.weather, 'day')
    expect(cloudy.state).toBe('cloudy')
    expect(stepWeather({ ...cloudy, until: 1 }, 2, 1, rng, tuning.weather, 'day').state).toBe('rain')
  })

  it('never selects snow in day but can select it at night', () => {
    const cloudy = { state: 'cloudy' as const, since: 0, until: 0, pressure: 1 }
    expect(stepWeather(cloudy, 1, 1, new Rng(2), tuning.weather, 'day').state).toBe('rain')
    expect(stepWeather(cloudy, 1, 1, new Rng(2), tuning.weather, 'night').state).toBe('snow')
  })
})

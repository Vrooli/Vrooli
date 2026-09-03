import { describe, expect, it } from 'vitest'
import { decideIntro } from './intro'

describe('decideIntro', () => {
  it('plays by default, skips under prefers-reduced-motion, and skips when the URL disables it', () => {
    expect(decideIntro(true, false)).toEqual({ play: true, reason: 'play' })
    expect(decideIntro(true, true)).toEqual({ play: false, reason: 'reduced-motion' })
    expect(decideIntro(false, false)).toEqual({ play: false, reason: 'disabled-by-url' })
    expect(decideIntro(false, true).play).toBe(false)
  })
})

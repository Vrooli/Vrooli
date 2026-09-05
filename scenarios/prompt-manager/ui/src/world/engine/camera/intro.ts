/**
 * Whether the establishing-to-hero dolly plays. Pure so the reduced-motion
 * rule is testable without a canvas.
 */
export interface IntroDecision {
  play: boolean
  reason: 'play' | 'disabled-by-url' | 'reduced-motion'
}

export function decideIntro(introRequested: boolean, reducedMotion: boolean): IntroDecision {
  if (!introRequested) return { play: false, reason: 'disabled-by-url' }
  if (reducedMotion) return { play: false, reason: 'reduced-motion' }
  return { play: true, reason: 'play' }
}

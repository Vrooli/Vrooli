import { beforeEach, describe, expect, it } from 'vitest'
import { READY_FRAMES, frameStats, readDiagnostics, recordFrame, resetDiagnostics, subscribeDiagnostics, updateDiagnostics } from './store'

describe('diagnostics store', () => {
  beforeEach(() => resetDiagnostics())

  it('flips ready only when assets, intro and the frame floor all hold', () => {
    updateDiagnostics({ assetsLoaded: true, introDone: true, framesRendered: READY_FRAMES - 1 })
    expect(readDiagnostics().ready).toBe(false)
    updateDiagnostics({ framesRendered: READY_FRAMES })
    expect(readDiagnostics().ready).toBe(true)
    updateDiagnostics({ introDone: false })
    expect(readDiagnostics().ready).toBe(false)
  })

  it('mirrors state onto window for the smoke tool and notifies subscribers', () => {
    let calls = 0
    const off = subscribeDiagnostics(() => { calls += 1 })
    updateDiagnostics({ drawCalls: 42 })
    expect(window.__worldDiagnostics?.drawCalls).toBe(42)
    expect(calls).toBe(1)
    off()
    updateDiagnostics({ drawCalls: 43 })
    expect(calls).toBe(1)
  })

  it('computes p50 and p95 over the recent frame window', () => {
    for (let i = 1; i <= 100; i += 1) recordFrame(i / 1000)
    const { p50, p95 } = frameStats()
    expect(p50).toBeCloseTo(50, 0)
    expect(p95).toBeCloseTo(95, 0)
  })
})

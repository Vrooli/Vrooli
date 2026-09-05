import { beforeEach, describe, expect, it } from 'vitest'
import { frameStats, readDiagnostics, recordFrame, resetDiagnostics, subscribeDiagnostics, updateDiagnostics } from './store'

describe('diagnostics store', () => {
  beforeEach(() => resetDiagnostics())

  it('uses the requested sample window and trims old samples when it shrinks', () => {
    for (let i = 1; i <= 10; i += 1) recordFrame(i / 1000, 10)
    recordFrame(0.011, 2)
    expect(frameStats()).toEqual({ p50: 10, p95: 10 })
    expect(readDiagnostics().totalFrames).toBe(11)
  })

  it('flips ready only when assets, intro and the frame floor all hold', () => {
    updateDiagnostics({ assetsLoaded: true, introDone: true, framesRendered: 11 })
    expect(readDiagnostics().ready).toBe(false)
    updateDiagnostics({ framesRendered: 12 })
    expect(readDiagnostics().ready).toBe(true)
    updateDiagnostics({ minimumReadyFps: 20 })
    expect(readDiagnostics().ready).toBe(false)
    updateDiagnostics({ framesRendered: 20 })
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
    expect(readDiagnostics().totalFrames).toBe(100)
    updateDiagnostics({ framesRendered: 30 })
    recordFrame(1 / 30)
    expect(readDiagnostics().totalFrames).toBe(101)
    expect(readDiagnostics().framesRendered).toBe(30)
    resetDiagnostics()
    expect(readDiagnostics().totalFrames).toBe(0)
  })
})

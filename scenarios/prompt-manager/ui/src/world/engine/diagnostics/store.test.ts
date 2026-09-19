import { beforeEach, describe, expect, it, vi } from 'vitest'
import { beginInteractionSample, endInteractionSample, recordInteractionInput, beginPresentation, beginTerrainPreparation, frameStats, readDiagnostics, recordFrame, recordPresentedFrame, resetDiagnostics, subscribeDiagnostics, updateDiagnostics } from './store'

describe('diagnostics store', () => {
  beforeEach(() => resetDiagnostics())
  it('waits for all terrain preparations and a frame after mesh attachment', () => {
    updateDiagnostics({ assetsLoaded: true, introDone: true })
    const first = beginTerrainPreparation()
    const second = beginTerrainPreparation()
    recordPresentedFrame()
    expect(readDiagnostics().ready).toBe(false)
    first()
    first()
    expect(readDiagnostics().terrainPreparations).toBe(1)
    second()
    expect(readDiagnostics().ready).toBe(false)
    recordPresentedFrame()
    expect(readDiagnostics().ready).toBe(true)
  })

  it('preserves the first readiness event through later rendering and readiness fluctuations', () => {
    const clock = vi.spyOn(performance, 'now').mockReturnValue(125)
    try {
      expect(readDiagnostics().firstReadyAt).toBeNull()
      updateDiagnostics({ assetsLoaded: true, introDone: true, framesRendered: 30 })
      recordPresentedFrame()
      expect(readDiagnostics().firstReadyAt).toBe(125)
      clock.mockReturnValue(5000)
      updateDiagnostics({ framesRendered: 0 })
      updateDiagnostics({ framesRendered: 30 })
      expect(readDiagnostics().firstReadyAt).toBe(125)
      resetDiagnostics()
      expect(readDiagnostics().firstReadyAt).toBeNull()
    } finally {
      clock.mockRestore()
    }
  })

  it('rejects old-frame readiness after a same-scene commit and timestamps the new epoch', () => {
    const clock = vi.spyOn(performance, 'now').mockReturnValue(100)
    try {
      beginPresentation(1)
      updateDiagnostics({ assetsLoaded: true, introDone: true })
      recordPresentedFrame(1)
      expect(readDiagnostics()).toMatchObject({ ready: true, epochReadyAt: 100, firstReadyAt: 100 })
      clock.mockReturnValue(200)
      beginPresentation(2)
      expect(readDiagnostics()).toMatchObject({ ready: false, epochReadyAt: null, presentationEpoch: 2 })
      updateDiagnostics({ assetsLoaded: true, introDone: true })
      recordPresentedFrame(1)
      expect(readDiagnostics().ready).toBe(false)
      beginPresentation(1)
      expect(readDiagnostics().presentationEpoch).toBe(2)
      recordPresentedFrame(2)
      expect(readDiagnostics()).toMatchObject({ ready: true, epochReadyAt: 200, firstReadyAt: 100 })
    } finally { clock.mockRestore() }
  })

  it('uses the requested sample window and trims old samples when it shrinks', () => {
    for (let i = 1; i <= 10; i += 1) recordFrame(i / 1000, 10)
    recordFrame(0.011, 2)
    expect(frameStats()).toEqual({ p50: 10, p95: 10 })
    expect(readDiagnostics().totalFrames).toBe(11)
  })

  it('becomes ready after presentation even at an intentional low idle cadence', () => {
    updateDiagnostics({ assetsLoaded: true, introDone: true, framesRendered: 4, minimumReadyFps: 12 })
    expect(readDiagnostics().ready).toBe(false)
    recordFrame(0.25)
    expect(readDiagnostics().ready).toBe(false)
    recordPresentedFrame()
    expect(readDiagnostics().ready).toBe(true)
    updateDiagnostics({ framesRendered: 0, minimumReadyFps: 60 })
    expect(readDiagnostics().ready).toBe(true)
  })

  it('requires a new presentation after loading, intro or scene changes', () => {
    recordPresentedFrame()
    expect(readDiagnostics().ready).toBe(false)
    updateDiagnostics({ assetsLoaded: true, introDone: true })
    recordPresentedFrame()
    expect(readDiagnostics().ready).toBe(true)
    for (const patch of [{ assetsLoaded: false }, { introDone: false }, { scene: 'office' as const }]) {
      updateDiagnostics(patch)
      expect(readDiagnostics().ready).toBe(false)
      updateDiagnostics({ assetsLoaded: true, introDone: true })
      expect(readDiagnostics().ready).toBe(false)
      recordPresentedFrame()
      expect(readDiagnostics().ready).toBe(true)
    }
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


it('samples only the requested interval and ignores the first idle frame gap', () => {
  const clock = vi.spyOn(performance, 'now').mockReturnValue(100)
  beginInteractionSample()
  recordFrame(20)
  recordInteractionInput()
  clock.mockReturnValue(112)
  recordFrame(0.016)
  recordPresentedFrame()
  updateDiagnostics({ cameraZoomGuard: { travel: 1, limited: false, queryMs: 0.4 } })
  const sample = endInteractionSample()!
  expect(sample.frames).toMatchObject({ count: 1, p95: 16, over50ms: 0 })
  expect(sample.inputToRenderMs).toMatchObject({ count: 1, p95: 12 })
  expect(sample.zoomQueryMs).toMatchObject({ count: 1, p95: 0.4 })
  expect(endInteractionSample()).toBeNull()
  clock.mockRestore()
})

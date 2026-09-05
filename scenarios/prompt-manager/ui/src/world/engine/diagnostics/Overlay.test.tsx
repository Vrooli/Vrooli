import { act, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { DiagnosticsOverlay } from './Overlay'
import { resetDiagnostics, updateDiagnostics } from './store'

afterEach(() => { vi.useRealTimers() })

it('refreshes at the configured cadence and replaces and clears its timer', () => {
  vi.useFakeTimers()
  resetDiagnostics()
  const view = render(<DiagnosticsOverlay seed={1} seedDigest="test" refreshMs={250} />)
  act(() => { updateDiagnostics({ drawCalls: 42 }); vi.advanceTimersByTime(249) })
  expect(screen.queryByText('42')).toBeNull()
  act(() => { vi.advanceTimersByTime(1) })
  expect(screen.getByText('42')).toBeTruthy()
  view.rerender(<DiagnosticsOverlay seed={1} seedDigest="test" refreshMs={500} />)
  act(() => { updateDiagnostics({ drawCalls: 43 }); vi.advanceTimersByTime(499) })
  expect(screen.queryByText('43')).toBeNull()
  act(() => { vi.advanceTimersByTime(1) })
  expect(screen.getByText('43')).toBeTruthy()
  expect(vi.getTimerCount()).toBe(1)
  view.unmount()
  expect(vi.getTimerCount()).toBe(0)
})

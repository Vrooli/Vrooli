import { act, renderWithProviders as render, screen } from '@/test-utils/renderWithProviders'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { WorkbenchStatus, type WorkbenchSnapshot } from './WorkbenchStatus'

afterEach(() => { vi.useRealTimers() })
describe('workbench runtime status', () => {
  it('shows stage outcomes and distinct cache estimates, refreshing only while open', () => {
    vi.useFakeTimers()
    const snapshot: WorkbenchSnapshot = {
      preparation: { epoch: 2, state: 'preparing', requestedAt: 10, completedAt: null,
        stages: [{ stage: 'assets', startedAt: 10, completedAt: 30, outcome: 'complete' }, { stage: 'generation', startedAt: 10, completedAt: null, outcome: 'running' }] },
      generation: { count: 1, source: 'worker', reason: 'initial' },
      caches: [{ name: 'Generated worlds', entries: 1, bytes: 1048576, budget: 67108864, hits: 0, misses: 1, evictions: 0 }],
    }
    const read = vi.fn(() => snapshot)
    const view = render(<WorkbenchStatus read={read} active={false} />)
    expect(read).not.toHaveBeenCalled()
    view.rerender(<WorkbenchStatus read={read} active />)
    expect(screen.getByRole('status')).toHaveTextContent('Preparation 2: preparing')
    expect(screen.getByText('assets: complete · 20 ms')).toBeInTheDocument()
    expect(screen.getByText('1.00 MiB / 64.00 MiB')).toBeInTheDocument()
    act(() => { vi.advanceTimersByTime(500) })
    expect(read).toHaveBeenCalledTimes(2)
    view.rerender(<WorkbenchStatus read={read} active={false} />)
    act(() => { vi.advanceTimersByTime(1000) })
    expect(read).toHaveBeenCalledTimes(2)
    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})

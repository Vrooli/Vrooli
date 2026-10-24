import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test-utils/renderWithProviders'
import { EventsDisplay, EventRow } from './EventsDisplay'

describe('EventsDisplay wire integration', () => {
  afterEach(() => { vi.unstubAllGlobals() })

  it('renders unspecified and typed variant events through the real Connect Value adapter', async () => {
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: RequestInfo | URL) => {
      const url = input instanceof Request ? input.url : String(input)
      if (!url.endsWith('.HeartbeatService/GetRunEvents')) throw new Error(`Unexpected request: ${url}`)
      const identity = { run_id: 'owner-run', timestamp: '2026-09-12T13:15:02.364059066Z' }
      return Promise.resolve(new Response(JSON.stringify({ data: { events: [
        { ...identity, id: 'unspecified', sequence: '8' },
        { ...identity, id: 'progress', sequence: '9', event_type: 'RUN_EVENT_TYPE_STATUS', progress: { phase: 'executing', percent: 5 } },
        { ...identity, id: 'future', sequence: '10', event_type: 'RUN_EVENT_TYPE_FUTURE', future: { proof: 'retained' } },
      ] } }), { status: 200, headers: { 'content-type': 'application/json' } }))
    }))
    render(<EventsDisplay runId="owner-run" />)
    expect(await screen.findByText('[unspecified]')).toBeInTheDocument()
    expect(screen.getByText(/"phase":"executing"/)).toBeInTheDocument()
    expect(screen.getByText(/"proof":"retained"/)).toBeInTheDocument()
    expect(screen.queryByText(/toLowerCase/)).not.toBeInTheDocument()
  })

  it('does not infer tool success from an omitted outcome', () => {
    render(<EventRow event={{
      id: 'tool', runId: 'owner-run', sequence: 2, timestamp: '2026-09-12T13:15:02Z',
      eventType: 'tool_result', data: { output: 'partial output' },
    }} />)
    expect(screen.getByText('Outcome unknown')).toBeInTheDocument()
    expect(screen.queryByText('Success')).not.toBeInTheDocument()
  })
})

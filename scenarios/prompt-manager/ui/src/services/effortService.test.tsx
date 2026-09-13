import { create } from '@bufbuild/protobuf'
import { EffortBoardSchema } from '@vrooli/proto-types/agent-manager/v1/domain/effort_pb'
import { afterEach, expect, test, vi } from 'vitest'
import { renderHookWithProviders } from '@/test'
import { waitFor } from '@/test-utils/renderWithProviders'
import { effortBoardClient, effortOwnerPath, readEffortObservations, useAgentManagerUrl } from './effortService'

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })

test('the embedded owner wire receives a bounded read and its cursor', async () => {
  const fetch = vi.fn().mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = input instanceof Request ? input.url : String(input)
    expect(url).toContain('/embedded/agent-manager/agent_manager.v1.AgentManagerService/GetEffortBoard')
    const body = input instanceof Request ? await input.text() : await new Response(init?.body).text()
    expect(JSON.parse(body)).toEqual({ pageSize: 10, pageToken: 'owner-cursor' })
    return new Response(JSON.stringify({ rows: [{ enrollment: { effortRef: 'effort:one' } }], nextPageToken: 'next-owner-cursor', partial: true }), { headers: { 'content-type': 'application/json' } })
  })
  vi.stubGlobal('fetch', fetch)
  const result = await readEffortObservations('owner-cursor')
  expect(result.boards[0]!.nextPageToken).toBe('next-owner-cursor')
  expect(result.boards[0]!.partial).toBe(true)
  expect(fetch).toHaveBeenCalledOnce()
})

test('exact linked reads retain partial failures and reject unrelated returned rows', async () => {
  const read = vi.spyOn(effortBoardClient, 'getEffortBoard')
    .mockResolvedValueOnce(create(EffortBoardSchema, { rows: [{ enrollment: { effortRef: 'one' } }, { enrollment: { effortRef: 'unrelated' } }] }))
    .mockRejectedValueOnce(new Error('owner offline'))
    .mockResolvedValueOnce(create(EffortBoardSchema, { rows: [{ enrollment: { effortRef: 'wrong-identity' } }] }))
  const result = await readEffortObservations('', ['one', 'two', 'three'])
  expect(read).toHaveBeenCalledTimes(3)
  expect(read).toHaveBeenNthCalledWith(2, { effortRef: 'two', pageSize: 1 }, { signal: undefined, timeoutMs: 30_000 })
  expect(result.boards.flatMap(board => board.rows).map(row => row.enrollment?.effortRef)).toEqual(['one'])
  expect(result.unavailable).toEqual([{ effortRef: 'two', reason: 'owner offline' }, { effortRef: 'three', reason: 'Exact effort was not returned by the owner' }])
})

test('requests beyond the linked page bound fail before reaching the owner', async () => {
  const read = vi.spyOn(effortBoardClient, 'getEffortBoard')
  await expect(readEffortObservations('', ['1', '2', '3', '4', '5', '6'])).rejects.toThrow('page limit')
  expect(read).not.toHaveBeenCalled()
})

test('outages remain errors and owner links retain exact effort identity', async () => {
  vi.spyOn(effortBoardClient, 'getEffortBoard').mockRejectedValueOnce(new Error('unavailable'))
  await expect(readEffortObservations('')).rejects.toThrow('unavailable')
  const reference = 'effort:alpha/beta?first=1&second=2'
  const url = new URL(effortOwnerPath('https://example.test/apps/agent-manager/proxy/', reference))
  expect(url.pathname).toBe('/apps/agent-manager/proxy/efforts')
  expect(url.searchParams.get('effortRef')).toBe(reference)
})

test('a closed helper does not request navigation, and opened owner links reject unsafe protocols', async () => {
  const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({ url: 'javascript:alert(1)' }), { headers: { 'content-type': 'application/json' } }))
  vi.stubGlobal('fetch', fetch)
  const { result, rerender } = renderHookWithProviders(({ enabled }) => useAgentManagerUrl(enabled), { initialProps: { enabled: false } })
  expect(fetch).not.toHaveBeenCalled()
  rerender({ enabled: true })
  await waitFor(() => expect(result.current.isError).toBe(true))
  expect(result.current.error?.message).toBe('Invalid Agent Manager link')
  expect(fetch).toHaveBeenCalledOnce()
})

import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { buildDefaultCreateTeamRequest, ValidationError } from '@/lib/schemas'
import { createTeam, getEffortWorkspaceContent, getTeams, invalidateCache } from './teamService'

beforeEach(() => { invalidateCache() })
afterEach(() => { vi.unstubAllGlobals(); invalidateCache() })

function ownerResponse(teams: unknown[]) {
  connectResponse('ListTeams', { teams })
}

function connectResponse(method: string, data: unknown) {
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async (input: RequestInfo | URL) => {
    const url = input instanceof Request ? input.url : String(input)
    if (!url.endsWith(`.TeamsService/${method}`)) throw new Error(`Unexpected owner request: ${url}`)
    return new Response(JSON.stringify(data), { headers: { 'content-type': 'application/json' } })
  }))
}

const team = () => ({
  ...buildDefaultCreateTeamRequest('Delivery'), id: 'delivery', enabled: false,
  memberCount: 0, createdAt: '2026-09-12T00:00:00Z', updatedAt: '2026-09-12T00:00:00Z',
})

test('invalid owner team data remains unavailable instead of a successful empty registry', async () => {
  ownerResponse([{ ...team(), purpose: 'unknown-purpose' }])
  const result = getTeams()
  await expect(result).rejects.toBeInstanceOf(ValidationError)
  await expect(result).rejects.toThrow(/purpose/)
  ownerResponse([])
  await expect(getTeams()).resolves.toEqual([])
})

test('owner-valid teams with more than 100 effort references remain readable', async () => {
  const effortRefs = Array.from({ length: 101 }, (_, i) => `effort:delivery/${i}`)
  ownerResponse([{ ...team(), effortRefs }])
  await expect(getTeams()).resolves.toEqual([expect.objectContaining({ id: 'delivery', memberCount: 0, effortRefs })])
})

test('creating an empty finite delivery team succeeds when Connect omits zero member count', async () => {
  connectResponse('CreateTeam', { ...team(), purpose: 'delivery', lifetime: 'finite', roles: [], members: [] })
  await expect(createTeam({ ...buildDefaultCreateTeamRequest('Delivery'), purpose: 'delivery', lifetime: 'finite' }))
    .resolves.toEqual(expect.objectContaining({ id: 'delivery', purpose: 'delivery', lifetime: 'finite', memberCount: 0, members: [], enabled: false }))
})

test('materializes bounded binary workspace bytes as a typed preview URL', async () => {
  connectResponse('GetEffortWorkspaceContent', {
    effortRef: 'effort:media', path: 'reference.png', content: '',
    previewDataUrl: 'data:image/png;base64,iVBORw0KGgo=',
  })
  const result = await getEffortWorkspaceContent('effort:media', 'reference.png')
  expect(result.content).toBe('data:image/png;base64,iVBORw0KGgo=')
})

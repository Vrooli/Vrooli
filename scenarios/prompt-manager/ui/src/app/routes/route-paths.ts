export type PromptManagerHomeRoute = 'world' | 'graph'
export type PromptManagerDetailEntity =
  | 'skill'
  | 'agent'
  | 'team'
  | 'run'
  | 'topic'
  | 'action'

export interface DetailRouteTarget {
  entityType: PromptManagerDetailEntity
  id: string
  query?: QueryParams
}

type QueryPrimitive = string | number | boolean | null | undefined
export type QueryParams = Record<string, QueryPrimitive>

const enc = encodeURIComponent

export function appendQuery(path: string, query?: QueryParams): string {
  if (!query) return path

  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value === null || value === undefined || value === '') continue
    params.set(key, String(value))
  }

  const serialized = params.toString()
  return serialized ? `${path}?${serialized}` : path
}

export function worldPath(query?: QueryParams): string {
  return appendQuery('/world', query)
}

export function graphPath(query?: QueryParams): string {
  return appendQuery('/graph', query)
}

export function skillDetailPath(skillId: string, query?: QueryParams): string {
  return appendQuery(`/skills/${enc(skillId)}`, query)
}

export function agentDetailPath(agentId: string, query?: QueryParams): string {
  return appendQuery(`/agents/${enc(agentId)}`, query)
}

export function teamDetailPath(teamId: string, query?: QueryParams): string {
  return appendQuery(`/teams/${enc(teamId)}`, query)
}

export function runDetailPath(runId: string, query?: QueryParams): string {
  return appendQuery(`/runs/${enc(runId)}`, query)
}

export function topicDetailPath(topicId: string, query?: QueryParams): string {
  return appendQuery(`/topics/${enc(topicId)}`, query)
}

export function actionDetailPath(actionId: string, query?: QueryParams): string {
  return appendQuery(`/actions/${enc(actionId)}`, query)
}

export function topicWizardPath(query?: QueryParams): string {
  return appendQuery('/topics/new', query)
}

export function homePath(route: PromptManagerHomeRoute, query?: QueryParams): string {
  return route === 'graph' ? graphPath(query) : worldPath(query)
}

export function detailPath(target: DetailRouteTarget): string {
  switch (target.entityType) {
    case 'skill':
      return skillDetailPath(target.id, target.query)
    case 'agent':
      return agentDetailPath(target.id, target.query)
    case 'team':
      return teamDetailPath(target.id, target.query)
    case 'run':
      return runDetailPath(target.id, target.query)
    case 'topic':
      return topicDetailPath(target.id, target.query)
    case 'action':
      return actionDetailPath(target.id, target.query)
  }
}

export function routeForEntity(
  entityType: 'skill' | 'agent' | 'team',
  entityId: string,
  query?: QueryParams
): string {
  if (entityType === 'agent') return agentDetailPath(entityId, query)
  if (entityType === 'team') return teamDetailPath(entityId, query)
  return skillDetailPath(entityId, query)
}

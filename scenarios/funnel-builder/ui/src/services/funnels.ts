import { Funnel, FunnelAnalytics, FunnelSettings, FunnelStep, Project } from '../types'
import { apiFetch } from '../utils/apiClient'

type RawJson = Record<string, unknown>

const defaultTheme = {
  primaryColor: '#0ea5e9',
  fontFamily: undefined,
  borderRadius: undefined
} as const

const toFunnelSettings = (input: unknown): FunnelSettings => {
  const candidate = input && typeof input === 'object' ? (input as RawJson) : {}
  const themeCandidate = candidate.theme && typeof candidate.theme === 'object'
    ? (candidate.theme as RawJson)
    : {}

  const trackingCandidate = candidate.tracking && typeof candidate.tracking === 'object'
    ? (candidate.tracking as RawJson)
    : undefined

  return {
    theme: {
      primaryColor: typeof themeCandidate.primaryColor === 'string'
        ? themeCandidate.primaryColor
        : defaultTheme.primaryColor,
      fontFamily: typeof themeCandidate.fontFamily === 'string'
        ? themeCandidate.fontFamily
        : defaultTheme.fontFamily,
      borderRadius: typeof themeCandidate.borderRadius === 'string'
        ? themeCandidate.borderRadius
        : defaultTheme.borderRadius
    },
    tracking: trackingCandidate
      ? {
          googleAnalytics: typeof trackingCandidate.googleAnalytics === 'string'
            ? trackingCandidate.googleAnalytics
            : undefined,
          facebookPixel: typeof trackingCandidate.facebookPixel === 'string'
            ? trackingCandidate.facebookPixel
            : undefined
        }
      : undefined,
    exitIntent: typeof candidate.exitIntent === 'boolean' ? candidate.exitIntent : undefined,
    progressBar: typeof candidate.progressBar === 'boolean' ? candidate.progressBar : true
  }
}

const toFunnelStep = (input: unknown, index: number): FunnelStep => {
  const candidate = input && typeof input === 'object' ? (input as RawJson) : {}

  const id = typeof candidate.id === 'string' ? candidate.id : `step-${index}`
  const type = typeof candidate.type === 'string' ? candidate.type : 'content'
  const title = typeof candidate.title === 'string' ? candidate.title : `Step ${index + 1}`
  const position = typeof candidate.position === 'number' ? candidate.position : index
  const content = candidate.content ?? {}

  const branchingRules = Array.isArray(candidate.branching_rules)
    ? candidate.branching_rules
    : Array.isArray(candidate.branchingRules)
      ? candidate.branchingRules
      : []

  return {
    id,
    type,
    position,
    title,
    content,
    branchingRules
  } as FunnelStep
}

const normalizeFunnel = (raw: unknown, index: number): Funnel => {
  const candidate = raw && typeof raw === 'object' ? (raw as RawJson) : {}

  const stepsSource = Array.isArray(candidate.steps) ? candidate.steps : []
  const steps = stepsSource
    .map((step, positionIndex) => toFunnelStep(step, positionIndex))
    .sort((a, b) => a.position - b.position)

  const projectId = typeof candidate.project_id === 'string'
    ? candidate.project_id
    : typeof candidate.projectId === 'string'
      ? candidate.projectId
      : null

  const createdAt = typeof candidate.created_at === 'string'
    ? candidate.created_at
    : typeof candidate.createdAt === 'string'
      ? candidate.createdAt
      : new Date().toISOString()

  const updatedAt = typeof candidate.updated_at === 'string'
    ? candidate.updated_at
    : typeof candidate.updatedAt === 'string'
      ? candidate.updatedAt
      : createdAt

  return {
    id: typeof candidate.id === 'string' ? candidate.id : `funnel-${index}`,
    tenantId: typeof candidate.tenant_id === 'string'
      ? candidate.tenant_id
      : typeof candidate.tenantId === 'string'
        ? candidate.tenantId
        : null,
    projectId,
    name: typeof candidate.name === 'string' ? candidate.name : 'Untitled Funnel',
    slug: typeof candidate.slug === 'string' ? candidate.slug : 'untitled-funnel',
    description: typeof candidate.description === 'string' ? candidate.description : undefined,
    steps,
    settings: toFunnelSettings(candidate.settings),
    createdAt,
    updatedAt,
    status: (candidate.status ?? 'draft') as Funnel['status']
  }
}

const normalizeProject = (raw: unknown, index: number): Project => {
  const candidate = raw && typeof raw === 'object' ? (raw as RawJson) : {}

  const funnelsSource = Array.isArray(candidate.funnels) ? candidate.funnels : []
  const funnels = funnelsSource.map((funnel, funnelIndex) => normalizeFunnel(funnel, funnelIndex))

  const createdAt = typeof candidate.created_at === 'string'
    ? candidate.created_at
    : typeof candidate.createdAt === 'string'
      ? candidate.createdAt
      : new Date().toISOString()

  const updatedAt = typeof candidate.updated_at === 'string'
    ? candidate.updated_at
    : typeof candidate.updatedAt === 'string'
      ? candidate.updatedAt
      : createdAt

  const descriptionRaw = typeof candidate.description === 'string' ? candidate.description.trim() : ''

  return {
    id: typeof candidate.id === 'string' ? candidate.id : `project-${index}`,
    tenantId: typeof candidate.tenant_id === 'string'
      ? candidate.tenant_id
      : typeof candidate.tenantId === 'string'
        ? candidate.tenantId
        : null,
    name: typeof candidate.name === 'string' && candidate.name.trim() !== ''
      ? candidate.name
      : 'Untitled Project',
    description: descriptionRaw !== '' ? descriptionRaw : undefined,
    createdAt,
    updatedAt,
    funnels
  }
}

export async function fetchFunnels(init?: RequestInit): Promise<Funnel[]> {
  const response = await apiFetch<unknown[]>('/funnels', init)
  return response.map((raw, index) => normalizeFunnel(raw, index))
}

export async function fetchFunnel(funnelId: string, init?: RequestInit): Promise<Funnel> {
  const response = await apiFetch<unknown>(`/funnels/${funnelId}`, init)
  return normalizeFunnel(response, 0)
}

export async function fetchFunnelAnalytics(
  funnelId: string,
  init?: RequestInit
): Promise<FunnelAnalytics> {
  return apiFetch<FunnelAnalytics>(`/funnels/${funnelId}/analytics`, init)
}

export interface ProjectPayload {
  name: string
  description?: string
  tenantId?: string
}

export async function fetchProjects(init?: RequestInit): Promise<Project[]> {
  const response = await apiFetch<unknown[]>('/projects', init)
  return response.map((raw, index) => normalizeProject(raw, index))
}

export interface CreateFunnelPayload {
  name: string
  description?: string
  slug?: string
  steps: FunnelStep[]
  settings: FunnelSettings
  status?: Funnel['status']
  tenantId?: string | null
  projectId: string
}

export async function createFunnel(payload: CreateFunnelPayload, init?: RequestInit): Promise<Funnel> {
  const bodyPayload: Record<string, unknown> = {
    name: payload.name,
    description: payload.description ?? '',
    slug: payload.slug,
    steps: payload.steps,
    settings: payload.settings,
    status: payload.status,
    project_id: payload.projectId,
  }

  if (payload.tenantId) {
    bodyPayload.tenant_id = payload.tenantId
  }

  const response = await apiFetch<{ id: string }>(
    '/funnels',
    {
      ...init,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(init?.headers ?? {})
      },
      body: JSON.stringify(bodyPayload)
    }
  )

  if (!response?.id) {
    throw new Error('Create funnel response missing id')
  }

  return fetchFunnel(response.id, init)
}

export interface UpdateFunnelPayload {
  name?: string
  description?: string
  steps?: FunnelStep[]
  settings?: FunnelSettings
  status?: Funnel['status']
  projectId?: string | null
}

export async function updateFunnel(
  funnelId: string,
  payload: UpdateFunnelPayload,
  init?: RequestInit
): Promise<Funnel> {
  const bodyPayload: Record<string, unknown> = {}

  if (payload.name !== undefined) {
    bodyPayload.name = payload.name
  }
  if (payload.description !== undefined) {
    bodyPayload.description = payload.description
  }
  if (payload.steps !== undefined) {
    bodyPayload.steps = payload.steps
  }
  if (payload.settings !== undefined) {
    bodyPayload.settings = payload.settings
  }
  if (payload.status !== undefined) {
    bodyPayload.status = payload.status
  }
  if (payload.projectId !== undefined) {
    bodyPayload.project_id = payload.projectId ?? null
  }

  await apiFetch<unknown>(
    `/funnels/${funnelId}`,
    {
      ...init,
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...(init?.headers ?? {})
      },
      body: JSON.stringify(bodyPayload)
    }
  )

  return fetchFunnel(funnelId, init)
}

export async function createProject(payload: ProjectPayload, init?: RequestInit): Promise<Project> {
  const body = JSON.stringify({
    name: payload.name,
    description: payload.description,
    tenant_id: payload.tenantId
  })

  return apiFetch<Project>('/projects', {
    ...init,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {})
    },
    body
  })
}

export async function updateProject(
  projectId: string,
  payload: Partial<ProjectPayload>,
  init?: RequestInit
): Promise<Project> {
  const body = JSON.stringify({
    name: payload.name,
    description: payload.description
  })

  return apiFetch<Project>(`/projects/${projectId}`, {
    ...init,
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {})
    },
    body
  })
}

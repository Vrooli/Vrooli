import type { DesignerEdge, GraphEdgeData } from '../types/graph'
import { EDGE_LABEL_STYLE, EDGE_LABEL_TEXT, EDGE_STYLE } from './constants'

export const isTemporaryEdgeId = (value: unknown): value is string =>
  typeof value === 'string' && value.startsWith('temp-')

export const generateTempEdgeId = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

interface CreateGraphEdgeInput {
  id: string
  source: string
  target: string
  dependencyType?: string
  dependencyStrength?: number
  description?: string
}

export const createGraphEdge = ({
  id,
  source,
  target,
  dependencyType,
  dependencyStrength,
  description
}: CreateGraphEdgeInput): DesignerEdge => {
  const resolvedType = dependencyType || 'required'
  const strength = typeof dependencyStrength === 'number' && !Number.isNaN(dependencyStrength)
    ? dependencyStrength
    : 1

  const data: GraphEdgeData = {
    dependencyType: resolvedType,
    dependencyStrength: strength,
    description: description || '',
    persistedId: !isTemporaryEdgeId(id) ? id : undefined
  }

  return {
    id,
    source,
    target,
    label: resolvedType.replace(/_/g, ' ') || EDGE_LABEL_TEXT,
    animated: strength >= 0.85,
    style: EDGE_STYLE,
    labelBgPadding: [4, 2],
    labelStyle: EDGE_LABEL_STYLE,
    data
  }
}

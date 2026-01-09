import type { CSSProperties } from 'react'
import type { ScenarioFormState, SectorFormState, StageFormState } from '../types/forms'

export type StageType =
  | 'foundation'
  | 'operational'
  | 'analytics'
  | 'integration'
  | 'digital_twin'
  | 'custom'

export const stageTypePalette: Record<StageType, string> = {
  foundation: '#10b981',
  operational: '#38bdf8',
  analytics: '#facc15',
  integration: '#a855f7',
  digital_twin: '#f97316',
  custom: '#60a5fa'
}

export const stageTypeLabel: Record<StageType, string> = {
  foundation: 'Foundation',
  operational: 'Operational',
  analytics: 'Analytics',
  integration: 'Integration',
  digital_twin: 'Digital Twin',
  custom: 'Custom'
}

export const sectorCategoryOptions: string[] = [
  'software',
  'individual',
  'manufacturing',
  'healthcare',
  'finance',
  'education',
  'governance',
  'science',
  'custom'
]

export type SectorCategory = (typeof sectorCategoryOptions)[number]

export const stageTypeOptions: StageType[] = [
  'foundation',
  'operational',
  'analytics',
  'integration',
  'digital_twin',
  'custom'
]

export const defaultSectorForm: SectorFormState = {
  name: '',
  category: 'software',
  description: '',
  color: '#0ea5e9',
  positionX: '',
  positionY: ''
}

export const defaultStageForm: StageFormState = {
  sectorId: '',
  stageType: 'foundation',
  stageOrder: '',
  name: '',
  description: '',
  progress: '0',
  positionX: '',
  positionY: '',
  examples: '',
  parentStageId: undefined
}

export const defaultScenarioForm: ScenarioFormState = {
  stageId: '',
  scenarioName: '',
  status: 'in_progress',
  contributionWeight: '1',
  priority: '3',
  estimatedImpact: '5',
  notes: ''
}

export const EDGE_STYLE: CSSProperties = {
  stroke: 'rgba(14, 165, 233, 0.45)',
  strokeWidth: 2
}

export const EDGE_LABEL_STYLE: CSSProperties = {
  fill: '#bae6fd',
  fontSize: 10,
  textTransform: 'uppercase',
  letterSpacing: 0.6
}

export const EDGE_LABEL_TEXT = 'dependency'

export const formatStageTypeLabel = (value?: string | null): string => {
  if (!value) return 'Stage'
  return value
    .split(/[_\s]+/)
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
}

export const getStageTypeColor = (value?: string): string => {
  if (!value) {
    return '#38bdf8'
  }
  const normalized = value as StageType
  return stageTypePalette[normalized] || '#38bdf8'
}

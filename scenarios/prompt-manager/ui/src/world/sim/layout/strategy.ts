import type { LayoutTuning } from '../../config'
import type { AgentInput, TeamInput } from '../model'
import { generateLayout, type GenerateOptions, type GeneratedLayout } from './generate'
import { floorplanStrategy } from './floorplan'

/** Shared input keeps optional biome data available without coupling callers to a strategy. */
export interface LayoutInput {
  teams: TeamInput[]
  agents: AgentInput[]
  tuning: LayoutTuning
  options: GenerateOptions
}

export interface LayoutStrategy {
  generate(input: LayoutInput): GeneratedLayout
}

export const clearingsStrategy: LayoutStrategy = {
  generate: ({ teams, agents, tuning, options }) => generateLayout(teams, agents, tuning, options),
}

export const layoutStrategies: Record<'clearings' | 'floorplan', LayoutStrategy | undefined> = {
  clearings: clearingsStrategy,
  floorplan: floorplanStrategy,
}

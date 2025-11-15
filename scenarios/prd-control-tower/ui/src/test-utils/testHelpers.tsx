import type { Draft, RequirementGroup, OperationalTarget } from '../types'
import { vi } from 'vitest'

/**
 * Test data factories for creating mock objects
 */

export function createMockDraft(overrides?: Partial<Draft>): Draft {
  return {
    id: 'test-draft-1',
    entity_type: 'scenario',
    entity_name: 'test-scenario',
    content: '# Test PRD\n\n## Overview\n\nTest content',
    owner: 'test-user',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    status: 'draft',
    ...overrides,
  }
}

export function createMockRequirementGroup(overrides?: Partial<RequirementGroup>): RequirementGroup {
  return {
    id: 'test-group',
    name: 'Test Group',
    description: 'Test description',
    requirements: [],
    children: [],
    ...overrides,
  }
}

export function createMockTarget(overrides?: Partial<OperationalTarget>): OperationalTarget {
  return {
    id: 'TEST-001',
    entity_type: 'scenario',
    entity_name: 'test-scenario',
    category: 'Must Have',
    criticality: 'P0',
    title: 'Test operational target',
    notes: 'Test notes',
    status: 'pending',
    path: 'Functional Requirements > Must Have',
    linked_requirement_ids: [],
    ...overrides,
  }
}

export function mockFetch(data: any, ok = true) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok,
    json: async () => data,
    statusText: ok ? 'OK' : 'Error',
  }) as any
}

export function mockFetchError(message: string) {
  globalThis.fetch = vi.fn().mockRejectedValue(new Error(message)) as any
}

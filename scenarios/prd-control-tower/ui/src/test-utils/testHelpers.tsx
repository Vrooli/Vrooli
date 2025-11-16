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

export function mockFetch<T>(data: T, ok = true): void {
  const mockResponse: Partial<Response> = {
    ok,
    status: ok ? 200 : 500,
    statusText: ok ? 'OK' : 'Error',
    json: async () => data,
    text: async () => JSON.stringify(data),
  }

  globalThis.fetch = vi.fn().mockResolvedValue(mockResponse as Response) as typeof fetch
}

export function mockFetchError(message: string): void {
  globalThis.fetch = vi.fn().mockRejectedValue(new Error(message)) as typeof fetch
}

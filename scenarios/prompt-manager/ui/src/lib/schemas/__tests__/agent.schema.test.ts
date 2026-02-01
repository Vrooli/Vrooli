import { describe, it, expect } from 'vitest'
import {
  AgentSchema,
  AgentArraySchema,
  AgentStatusSchema,
  AgentAppearanceSchema,
  CreateAgentRequestSchema,
  EffectiveSkillsResponseSchema,
  DEFAULT_AGENT_COLORS,
} from '../agent.schema'

describe('AgentStatusSchema', () => {
  it('should accept valid status values', () => {
    expect(AgentStatusSchema.parse('active')).toBe('active')
    expect(AgentStatusSchema.parse('inactive')).toBe('inactive')
    expect(AgentStatusSchema.parse('suspended')).toBe('suspended')
  })

  it('should reject invalid status values', () => {
    expect(AgentStatusSchema.safeParse('invalid').success).toBe(false)
    expect(AgentStatusSchema.safeParse('').success).toBe(false)
    expect(AgentStatusSchema.safeParse('ACTIVE').success).toBe(false)
  })
})

describe('AgentAppearanceSchema', () => {
  it('should parse valid hex colors', () => {
    const appearance = {
      body: '#FF5733',
      head: '#33FF57',
      accent: '#3357FF',
    }

    const result = AgentAppearanceSchema.parse(appearance)

    expect(result.body).toBe('#FF5733')
    expect(result.head).toBe('#33FF57')
    expect(result.accent).toBe('#3357FF')
  })

  it('should accept lowercase hex colors', () => {
    const appearance = {
      body: '#ff5733',
      head: '#33ff57',
      accent: '#3357ff',
    }

    expect(AgentAppearanceSchema.safeParse(appearance).success).toBe(true)
  })

  it('should reject invalid hex colors', () => {
    expect(
      AgentAppearanceSchema.safeParse({
        body: 'red',
        head: '#33FF57',
        accent: '#3357FF',
      }).success
    ).toBe(false)

    expect(
      AgentAppearanceSchema.safeParse({
        body: '#FFF', // 3-digit not allowed
        head: '#33FF57',
        accent: '#3357FF',
      }).success
    ).toBe(false)

    expect(
      AgentAppearanceSchema.safeParse({
        body: 'FF5733', // Missing #
        head: '#33FF57',
        accent: '#3357FF',
      }).success
    ).toBe(false)
  })
})

describe('AgentSchema', () => {
  const minimalAgent = {
    id: 'agent-1',
    displayName: 'Test Agent',
    status: 'active',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  it('should parse a minimal agent with skills default', () => {
    const result = AgentSchema.parse(minimalAgent)

    expect(result.id).toBe('agent-1')
    expect(result.displayName).toBe('Test Agent')
    expect(result.status).toBe('active')
    // Verify default for skills array - prevents crashes
    expect(result.skills).toEqual([])
  })

  it('should provide default for skills array - prevents crashes', () => {
    // This tests the key safety feature: when API omits skills, we get [] not undefined
    const agent = AgentSchema.parse(minimalAgent)

    expect(Array.isArray(agent.skills)).toBe(true)
    expect(agent.skills.length).toBe(0)

    // Safe to iterate without crashing
    expect(() => agent.skills.map((s) => s.toUpperCase())).not.toThrow()
    expect(() => agent.skills.filter((s) => s.includes('x'))).not.toThrow()
  })

  it('should preserve provided skills array', () => {
    const agentWithSkills = {
      ...minimalAgent,
      skills: ['skill-1', 'skill-2', 'skill-3'],
    }

    const result = AgentSchema.parse(agentWithSkills)

    expect(result.skills).toEqual(['skill-1', 'skill-2', 'skill-3'])
  })

  it('should parse agent with appearance', () => {
    const agentWithAppearance = {
      ...minimalAgent,
      appearance: {
        body: '#4f46e5',
        head: '#818cf8',
        accent: '#c7d2fe',
      },
    }

    const result = AgentSchema.parse(agentWithAppearance)

    expect(result.appearance).toEqual({
      body: '#4f46e5',
      head: '#818cf8',
      accent: '#c7d2fe',
    })
  })

  it('should reject missing required fields', () => {
    expect(AgentSchema.safeParse({}).success).toBe(false)
    expect(AgentSchema.safeParse({ id: 'test' }).success).toBe(false)
    expect(
      AgentSchema.safeParse({ id: 'test', displayName: 'Test' }).success
    ).toBe(false)
  })

  it('should reject invalid status', () => {
    const invalidAgent = {
      ...minimalAgent,
      status: 'invalid',
    }

    expect(AgentSchema.safeParse(invalidAgent).success).toBe(false)
  })
})

describe('AgentArraySchema', () => {
  it('should parse an array of agents', () => {
    const agents = [
      {
        id: 'agent-1',
        displayName: 'Agent 1',
        status: 'active',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
      {
        id: 'agent-2',
        displayName: 'Agent 2',
        status: 'inactive',
        skills: ['skill-1'],
        createdAt: '2024-01-02T00:00:00Z',
        updatedAt: '2024-01-02T00:00:00Z',
      },
    ]

    const result = AgentArraySchema.parse(agents)

    expect(result).toHaveLength(2)
    expect(result[0]?.id).toBe('agent-1')
    expect(result[0]?.skills).toEqual([]) // Default applied
    expect(result[1]?.id).toBe('agent-2')
    expect(result[1]?.skills).toEqual(['skill-1']) // Preserved
  })

  it('should parse empty array', () => {
    const result = AgentArraySchema.parse([])
    expect(result).toEqual([])
  })
})

describe('CreateAgentRequestSchema', () => {
  it('should validate a valid create request', () => {
    const request = {
      displayName: 'New Agent',
    }

    const result = CreateAgentRequestSchema.parse(request)

    expect(result.displayName).toBe('New Agent')
  })

  it('should validate with optional fields', () => {
    const request = {
      displayName: 'New Agent',
      appearance: DEFAULT_AGENT_COLORS,
      skills: ['skill-1'],
    }

    const result = CreateAgentRequestSchema.parse(request)

    expect(result.displayName).toBe('New Agent')
    expect(result.appearance).toEqual(DEFAULT_AGENT_COLORS)
    expect(result.skills).toEqual(['skill-1'])
  })

  it('should reject empty displayName', () => {
    const request = {
      displayName: '',
    }

    const parseResult = CreateAgentRequestSchema.safeParse(request)
    expect(parseResult.success).toBe(false)
  })

  it('should reject displayName over 100 characters', () => {
    const request = {
      displayName: 'A'.repeat(101),
    }

    const parseResult = CreateAgentRequestSchema.safeParse(request)
    expect(parseResult.success).toBe(false)
  })
})

describe('EffectiveSkillsResponseSchema', () => {
  it('should parse minimal response with skills default', () => {
    const response = {
      agentId: 'agent-1',
    }

    const result = EffectiveSkillsResponseSchema.parse(response)

    expect(result.agentId).toBe('agent-1')
    expect(result.skills).toEqual([]) // Default applied
  })

  it('should preserve provided skills', () => {
    const response = {
      agentId: 'agent-1',
      teamId: 'team-1',
      skills: ['skill-a', 'skill-b'],
    }

    const result = EffectiveSkillsResponseSchema.parse(response)

    expect(result.agentId).toBe('agent-1')
    expect(result.teamId).toBe('team-1')
    expect(result.skills).toEqual(['skill-a', 'skill-b'])
  })
})

describe('DEFAULT_AGENT_COLORS', () => {
  it('should be valid AgentAppearance', () => {
    expect(AgentAppearanceSchema.safeParse(DEFAULT_AGENT_COLORS).success).toBe(
      true
    )
  })

  it('should have expected indigo color palette', () => {
    expect(DEFAULT_AGENT_COLORS.body).toBe('#4f46e5')
    expect(DEFAULT_AGENT_COLORS.head).toBe('#818cf8')
    expect(DEFAULT_AGENT_COLORS.accent).toBe('#c7d2fe')
  })
})

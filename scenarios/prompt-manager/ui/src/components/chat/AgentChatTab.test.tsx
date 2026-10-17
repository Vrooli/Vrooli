import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@/test-utils/renderWithProviders'
import { renderWithProviders } from '@/test-utils/renderWithProviders'
import { AgentChatTab } from './AgentChatTab'
import type { SessionStorage } from './conversationSession'
import {
  createMemberConversation,
  getRunDetails,
  getRunEvents,
  newConversationRequestId,
  type RunDetails,
} from '@/services/heartbeatService'

vi.mock('@/world/data/roster', () => ({
  useWorldRoster: () => ({
    ready: true,
    agents: [
      { id: 'ada', name: 'Ada', skillCount: 0 },
      { id: 'bob', name: 'Bob', skillCount: 0 },
    ],
    teams: [
      { id: 'team-a', name: 'Marketing', memberIds: ['ada'] },
      { id: 'team-b', name: 'Research', memberIds: ['ada'] },
      { id: 'team-c', name: 'Ops', memberIds: ['bob'] },
    ],
  }),
}))

vi.mock('@/services/heartbeatService', () => ({
  continueRun: vi.fn(),
  createMemberConversation: vi.fn(),
  getRunDetails: vi.fn(),
  getRunEvents: vi.fn(),
  newConversationRequestId: vi.fn(),
}))

function memoryStorage(): SessionStorage {
  const map = new Map<string, string>()
  return {
    getItem: (key) => (map.has(key) ? (map.get(key) as string) : null),
    setItem: (key, value) => {
      map.set(key, value)
    },
    removeItem: (key) => {
      map.delete(key)
    },
  }
}

function makeRun(overrides: Partial<RunDetails> = {}): RunDetails {
  return {
    id: 'run-1',
    taskId: 'task-1',
    status: 'completed',
    actions: {
      canInvestigate: false,
      canApplyInvestigation: false,
      canDelete: false,
      canStop: false,
      canRetry: false,
      canContinue: true,
    },
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getRunDetails).mockResolvedValue(makeRun())
  vi.mocked(getRunEvents).mockResolvedValue([])
})

describe('AgentChatTab', () => {
  it('defaults to the agent base context and offers only that agent teams', () => {
    renderWithProviders(<AgentChatTab agentId="ada" storage={memoryStorage()} />)

    expect(screen.getByRole('combobox', { name: 'Agent' })).toHaveValue('ada')
    expect(screen.getByTestId('conversation-context')).toHaveTextContent('Base agent')
    expect(screen.getByRole('option', { name: 'Marketing' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Research' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Ops' })).not.toBeInTheDocument()
  })

  it('keeps the selected team-member context visible when it changes', () => {
    renderWithProviders(<AgentChatTab agentId="ada" storage={memoryStorage()} />)

    fireEvent.change(screen.getByRole('combobox', { name: 'Context' }), {
      target: { value: 'team-a' },
    })

    expect(screen.getByTestId('conversation-context')).toHaveTextContent('Marketing')
  })

  it('starts the first turn against the selected team-member context', async () => {
    vi.mocked(newConversationRequestId).mockReturnValue('req-1')
    vi.mocked(createMemberConversation).mockResolvedValue(makeRun({ id: 'run-9', status: 'running' }))

    renderWithProviders(<AgentChatTab agentId="ada" storage={memoryStorage()} />)

    fireEvent.change(screen.getByRole('combobox', { name: 'Context' }), {
      target: { value: 'team-a' },
    })
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'hello' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }))

    await waitFor(() =>
      expect(createMemberConversation).toHaveBeenCalledWith({
        teamId: 'team-a',
        agentId: 'ada',
        message: 'hello',
        requestId: 'req-1',
      }),
    )
  })

  it('still offers the page agent when the roster has not loaded it', () => {
    renderWithProviders(<AgentChatTab agentId="ghost" storage={memoryStorage()} />)
    expect(screen.getByRole('option', { name: 'ghost' })).toBeInTheDocument()
  })

  it('hides the inline selector and reports changes when the context is controlled', () => {
    const onIdentityChange = vi.fn()
    renderWithProviders(
      <AgentChatTab
        agentId="ada"
        identity={{ agentId: 'ada', teamId: 'team-a' }}
        onIdentityChange={onIdentityChange}
        showContextSelector={false}
        storage={memoryStorage()}
      />,
    )

    expect(screen.queryByRole('combobox', { name: 'Context' })).not.toBeInTheDocument()
    expect(screen.getByTestId('conversation-context')).toHaveTextContent('Marketing')
    expect(onIdentityChange).not.toHaveBeenCalled()
  })
})

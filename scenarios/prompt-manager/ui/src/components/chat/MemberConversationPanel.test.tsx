import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@/test-utils/renderWithProviders'
import { renderWithProviders } from '@/test-utils/renderWithProviders'
import { MemberConversationPanel } from './MemberConversationPanel'
import {
  createConversationSession,
  writeConversationSession,
  type SessionStorage,
} from './conversationSession'
import {
  continueRun,
  createMemberConversation,
  getRunDetails,
  getRunEvents,
  newConversationRequestId,
  type RunDetails,
} from '@/services/heartbeatService'

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

const agents = [{ id: 'ada', name: 'Ada' }]
const memberships = [{ teamId: 'team-a', teamName: 'Marketing', agentId: 'ada' }]
const identity = { agentId: 'ada', teamId: 'team-a' }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getRunDetails).mockResolvedValue(makeRun())
  vi.mocked(getRunEvents).mockResolvedValue([])
})

describe('MemberConversationPanel', () => {
  it('shows the selected context and starts the first turn', async () => {
    vi.mocked(newConversationRequestId).mockReturnValue('req-start')
    vi.mocked(createMemberConversation).mockResolvedValue(makeRun({ id: 'run-9', status: 'running' }))
    const storage = memoryStorage()

    renderWithProviders(
      <MemberConversationPanel
        agents={agents}
        memberships={memberships}
        identity={identity}
        onIdentityChange={vi.fn()}
        storage={storage}
      />,
    )

    expect(screen.getByTestId('conversation-context')).toHaveTextContent('Marketing')
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'hello' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }))

    await waitFor(() =>
      expect(createMemberConversation).toHaveBeenCalledWith({
        teamId: 'team-a',
        agentId: 'ada',
        message: 'hello',
        requestId: 'req-start',
      }),
    )
  })

  it('labels a failed start as a start recovery', async () => {
    vi.mocked(newConversationRequestId).mockReturnValue('req-start')
    vi.mocked(createMemberConversation).mockRejectedValue(new Error('response lost'))

    renderWithProviders(
      <MemberConversationPanel
        agents={agents}
        memberships={memberships}
        identity={identity}
        onIdentityChange={vi.fn()}
        storage={memoryStorage()}
      />,
    )

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'hello' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }))

    const recovery = await screen.findByTestId('conversation-recovery')
    expect(recovery).toHaveAttribute('data-operation', 'start')
  })

  it('labels a failed turn as a continue recovery', async () => {
    const storage = memoryStorage()
    writeConversationSession(createConversationSession(identity, 'run-1', 'req-first'), storage)
    vi.mocked(newConversationRequestId).mockReturnValue('turn-1')
    vi.mocked(continueRun).mockRejectedValue(new Error('network'))

    renderWithProviders(
      <MemberConversationPanel
        agents={agents}
        memberships={memberships}
        identity={identity}
        onIdentityChange={vi.fn()}
        storage={storage}
      />,
    )

    await waitFor(() => expect(getRunDetails).toHaveBeenCalledWith('run-1'))
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'second' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }))

    const recovery = await screen.findByTestId('conversation-recovery')
    expect(recovery).toHaveAttribute('data-operation', 'continue')
    expect(continueRun).toHaveBeenCalledWith('run-1', 'second', { requestId: 'turn-1' })
  })
})

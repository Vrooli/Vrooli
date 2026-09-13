import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@/test-utils/renderWithProviders'
import { renderWithProviders } from '@/test-utils/renderWithProviders'
import { AgentEditorPanel } from './AgentEditorPanel'
import type { Agent } from '@/types/agent'

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

vi.mock('@/services/agentService', () => ({
  getAgentTeams: vi.fn(() =>
    Promise.resolve([
      { teamId: 'team-a', teamDisplayName: 'Marketing' },
      { teamId: 'team-b', teamDisplayName: 'Research' },
    ]),
  ),
  previewAgentPromptStructured: vi.fn(() =>
    Promise.resolve({ sections: [{ kind: 'agent', label: 'Role', content: 'You are Ada.' }] }),
  ),
}))

vi.mock('@/services/heartbeatService', () => ({
  continueRun: vi.fn(),
  createMemberConversation: vi.fn(),
  getRunDetails: vi.fn(),
  getRunEvents: vi.fn(),
  newConversationRequestId: vi.fn(),
}))

import * as agentService from '@/services/agentService'

const agent: Agent = {
  id: 'ada',
  displayName: 'Ada',
  description: '',
  status: 'active',
  appearance: null,
  capabilities: null,
  connectors: [],
  tags: [],
  fileOrder: [],
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

function renderPanel() {
  return renderWithProviders(
    <AgentEditorPanel
      agent={agent}
      formState={{
        displayName: 'Ada',
        description: '',
        status: 'active',
        appearance: { body: '#000', head: '#fff', accent: '#f00' },
        tags: [],
        fileOrder: [],
      }}
      updateField={vi.fn()}
      updateFields={vi.fn()}
      renameFileOrderPath={vi.fn()}
      validation={{ valid: true, errors: {} }}
      isDirty={false}
      dirtyCount={0}
      onUndo={vi.fn()}
      onRedo={vi.fn()}
      canUndo={false}
      canRedo={false}
      onSave={vi.fn()}
      onSaveAll={vi.fn()}
      onDiscard={vi.fn()}
      onDelete={vi.fn()}
      onDuplicate={vi.fn()}
      onClose={vi.fn()}
      initialTab="prompt"
      enableChat
    />,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('AgentEditorPanel shared persona context', () => {
  it('shows the shared context selector on the prompt tab', async () => {
    renderPanel()

    expect(screen.getByTestId('agent-editor-shared-context')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Agent' })).toHaveValue('ada')
    expect(screen.getByRole('option', { name: 'Marketing' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Research' })).toBeInTheDocument()

    await waitFor(() =>
      expect(agentService.previewAgentPromptStructured).toHaveBeenCalledWith('ada', undefined),
    )
  })

  it('moves the prompt preview to the selected team context', async () => {
    renderPanel()

    await waitFor(() =>
      expect(agentService.previewAgentPromptStructured).toHaveBeenCalledWith('ada', undefined),
    )

    fireEvent.change(screen.getByRole('combobox', { name: 'Context' }), {
      target: { value: 'team-a' },
    })

    await waitFor(() =>
      expect(agentService.previewAgentPromptStructured).toHaveBeenCalledWith('ada', 'team-a'),
    )
  })
})

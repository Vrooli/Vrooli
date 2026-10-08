import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test-utils/renderWithProviders'
import { Code, ConnectError } from '@connectrpc/connect'
import { FilesTab } from './FilesTab'
import * as agentService from '@/services/agentService'
import { createEmptyAgentState } from '@/stores/agentEditorStore'

vi.mock('@/services/agentService', async () => {
  const actual = await vi.importActual<typeof import('@/services/agentService')>('@/services/agentService')
  return {
    ...actual,
    listAgentFiles: vi.fn(),
    listAgentFileTemplates: vi.fn(),
    getAgentFileContent: vi.fn(),
    setAgentFileContent: vi.fn(),
  }
})

vi.mock('@/hooks/use-toast', () => ({
  toast: vi.fn(),
  useToast: () => ({ toasts: [], toast: vi.fn(), dismiss: vi.fn() }),
}))

vi.mock('../SkillContentEditor', () => ({
  SkillContentEditor: ({ value, onChange }: { value: string; onChange: (next: string) => void }) => (
    <textarea
      aria-label="file-content"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  ),
  EditorActionButtons: () => null,
}))

const listAgentFiles = vi.mocked(agentService.listAgentFiles)
const listAgentFileTemplates = vi.mocked(agentService.listAgentFileTemplates)
const getAgentFileContent = vi.mocked(agentService.getAgentFileContent)

function renderFilesTab(agentId: string) {
  const props = {
    agentId,
    formState: createEmptyAgentState(),
    updateField: vi.fn(),
    updateFields: vi.fn(),
    renameFileOrderPath: vi.fn(),
    isDirty: false,
    dirtyCount: 0,
    onUndo: vi.fn(),
    onRedo: vi.fn(),
    canUndo: false,
    canRedo: false,
    onSave: vi.fn(),
    onDiscard: vi.fn(),
    isSaving: false,
    isValid: true,
  }
  return render(<FilesTab {...props} />)
}

beforeEach(() => {
  vi.clearAllMocks()
  listAgentFileTemplates.mockResolvedValue([])
  getAgentFileContent.mockResolvedValue('hello')
})

describe('FilesTab file states (R27)', () => {
  it('renders a populated listing', async () => {
    listAgentFiles.mockResolvedValue([
      { path: 'SOUL.md', isDir: false, size: 10 },
      { path: 'AGENTS.md', isDir: false, size: 12 },
    ])

    renderFilesTab('brand-manager')

    expect(await screen.findByText('SOUL.md')).toBeInTheDocument()
    expect(screen.getByText('AGENTS.md')).toBeInTheDocument()
    expect(screen.queryByText(/No files yet/)).not.toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('renders an empty listing distinctly from a failure', async () => {
    listAgentFiles.mockResolvedValue([])

    renderFilesTab('brand-manager')

    expect(await screen.findByText(/No files yet/)).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('renders a missing agent as a distinct error with retry', async () => {
    listAgentFiles.mockRejectedValue(new ConnectError('not found', Code.NotFound))

    renderFilesTab('deleted-agent')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Agent not found')
    expect(screen.getByRole('button', { name: /Retry/ })).toBeInTheDocument()
    expect(screen.queryByText(/No files yet/)).not.toBeInTheDocument()
  })

  it('renders denied access distinctly', async () => {
    listAgentFiles.mockRejectedValue(new ConnectError('denied', Code.PermissionDenied))

    renderFilesTab('private-agent')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Files access denied')
  })

  it('renders an unavailable server distinctly from an empty listing', async () => {
    listAgentFiles.mockRejectedValue(new ConnectError('offline', Code.Unavailable))

    renderFilesTab('brand-manager')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Files unavailable')
    expect(screen.queryByText(/No files yet/)).not.toBeInTheDocument()
  })
})

describe('FilesTab recovery and isolation (R27)', () => {
  it('preserves selection and unsaved edits across a failed refresh and retry', async () => {
    listAgentFiles.mockResolvedValue([{ path: 'SOUL.md', isDir: false, size: 10 }])

    renderFilesTab('brand-manager')

    const editor = await screen.findByLabelText('file-content')
    await waitFor(() => expect(editor).toHaveValue('hello'))
    fireEvent.change(editor, { target: { value: 'hello world' } })
    expect(editor).toHaveValue('hello world')

    listAgentFiles.mockRejectedValueOnce(new ConnectError('offline', Code.Unavailable))
    fireEvent.click(screen.getByTitle('Refresh'))

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('Files unavailable')
    expect(screen.getByText('SOUL.md')).toBeInTheDocument()
    expect(screen.getByLabelText('file-content')).toHaveValue('hello world')

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(screen.getByLabelText('file-content')).toHaveValue('hello world')
    })
    expect(screen.getByText('SOUL.md')).toBeInTheDocument()
  })

  it('never shows another agent\u2019s files after agent navigation', async () => {
    listAgentFiles.mockImplementation((id: string) =>
      Promise.resolve(
        id === 'agent-a'
          ? [{ path: 'a-only.md', isDir: false, size: 1 }]
          : [{ path: 'b-only.md', isDir: false, size: 2 }]
      )
    )
    getAgentFileContent.mockImplementation((_id: string, path: string) =>
      Promise.resolve(path === 'a-only.md' ? 'content-a' : 'content-b')
    )

    const view = renderFilesTab('agent-a')
    expect(await screen.findByText('a-only.md')).toBeInTheDocument()

    view.rerender(
      <FilesTab
        agentId="agent-b"
        formState={createEmptyAgentState()}
        updateField={vi.fn()}
        updateFields={vi.fn()}
        renameFileOrderPath={vi.fn()}
        isDirty={false}
        dirtyCount={0}
        onUndo={vi.fn()}
        onRedo={vi.fn()}
        canUndo={false}
        canRedo={false}
        onSave={vi.fn()}
        onDiscard={vi.fn()}
        isSaving={false}
        isValid
      />
    )

    expect(await screen.findByText('b-only.md')).toBeInTheDocument()
    expect(screen.queryByText('a-only.md')).not.toBeInTheDocument()
    expect(listAgentFiles).toHaveBeenLastCalledWith('agent-b')
    await waitFor(() => expect(screen.getByLabelText('file-content')).toHaveValue('content-b'))
  })
})

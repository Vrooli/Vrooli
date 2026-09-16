import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@/test-utils/renderWithProviders'
import { TeamFilesTab } from './TeamFilesTab'
import * as teamService from '@/services/teamService'

vi.mock('@/services/teamService', async () => {
  const actual = await vi.importActual<typeof import('@/services/teamService')>('@/services/teamService')
  return {
    ...actual,
    listTeamSharedFiles: vi.fn(),
    getTeamSharedFileContent: vi.fn(),
    setTeamSharedFileContent: vi.fn(),
    listEffortWorkspaces: vi.fn(),
    getEffortWorkspaceContent: vi.fn(),
  }
})

vi.mock('@/hooks/use-toast', () => ({
  toast: vi.fn(),
  useToast: () => ({ toasts: [], toast: vi.fn(), dismiss: vi.fn() }),
}))

vi.mock('../SkillContentEditor', () => ({
  SkillContentEditor: ({ value }: { value: string }) => (
    <textarea aria-label="team-file-content" value={value} readOnly />
  ),
}))

const listTeamSharedFiles = vi.mocked(teamService.listTeamSharedFiles)
const getTeamSharedFileContent = vi.mocked(teamService.getTeamSharedFileContent)
const listEffortWorkspaces = vi.mocked(teamService.listEffortWorkspaces)
const getEffortWorkspaceContent = vi.mocked(teamService.getEffortWorkspaceContent)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('TeamFilesTab isolation (R27)', () => {
  it('never shows another team\u2019s shared files after navigation', async () => {
    listTeamSharedFiles.mockImplementation((teamId: string) =>
      Promise.resolve(
        teamId === 'team-a'
          ? [{ path: 'a-shared.md', isDir: false, size: 1 }]
          : [{ path: 'b-shared.md', isDir: false, size: 2 }]
      )
    )
    getTeamSharedFileContent.mockImplementation((_teamId: string, path: string) =>
      Promise.resolve(path === 'a-shared.md' ? 'team-a-content' : 'team-b-content')
    )

    const view = render(<TeamFilesTab teamId="team-a" />)
    expect(await screen.findByText('a-shared.md')).toBeInTheDocument()

    view.rerender(<TeamFilesTab teamId="team-b" />)

    expect(await screen.findByText('b-shared.md')).toBeInTheDocument()
    expect(screen.queryByText('a-shared.md')).not.toBeInTheDocument()
    expect(listTeamSharedFiles).toHaveBeenLastCalledWith('team-b')
    await waitFor(() =>
      expect(screen.getByLabelText('team-file-content')).toHaveValue('team-b-content')
    )
  })

  it('shows eligible effort files in the same sidebar as team files', async () => {
    listEffortWorkspaces.mockResolvedValue({
      teamId: 'team-a',
      workspaces: [{ effortRef: 'effort:one', slug: 'one', stage: 'intake', files: [{ path: 'README.md', isDir: false, size: 4 }] }],
      unavailable: [],
    })
    getEffortWorkspaceContent.mockResolvedValue({ effortRef: 'effort:one', path: 'README.md', content: 'workspace-read' })

    render(<TeamFilesTab teamId="team-a" showEffortWorkspaces />)

    expect(await screen.findByText('one')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'README.md' }))
    expect(await screen.findByText('workspace-read')).toBeInTheDocument()
    expect(getEffortWorkspaceContent).toHaveBeenCalledWith('effort:one', 'README.md')
  })

  it('does not load effort files for ineligible teams', async () => {
    render(<TeamFilesTab teamId="standing-team" />)

    await waitFor(() => expect(listEffortWorkspaces).not.toHaveBeenCalled())
    expect(screen.queryByText('Effort workspaces')).not.toBeInTheDocument()
  })

  it('renders linked Markdown with an explicit read-only artifact header', async () => {
    listEffortWorkspaces.mockResolvedValue({
      teamId: 'team-a',
      workspaces: [{ effortRef: 'effort:one', slug: 'one', stage: 'review', files: [{ path: 'README.md', isDir: false, size: 12 }] }],
      unavailable: [],
    })
    getEffortWorkspaceContent.mockResolvedValue({ effortRef: 'effort:one', path: 'README.md', content: '# Campaign brief' })

    render(<TeamFilesTab teamId="team-a" showEffortWorkspaces />)

    fireEvent.click(await screen.findByRole('button', { name: 'README.md' }))
    expect(await screen.findByTestId('team-file-artifact-preview')).toBeInTheDocument()
    expect(screen.getByText(/Markdown/)).toBeInTheDocument()
    expect(screen.getByText('Read only')).toBeInTheDocument()
    expect(screen.getByText('Campaign brief')).toBeInTheDocument()
  })

  it('labels binary linked artifacts honestly when bytes are unavailable', async () => {
    listEffortWorkspaces.mockResolvedValue({
      teamId: 'team-a',
      workspaces: [{ effortRef: 'effort:one', slug: 'one', stage: 'review', files: [{ path: 'reference.png', isDir: false, size: 2048 }] }],
      unavailable: [],
    })
    getEffortWorkspaceContent.mockResolvedValue({ effortRef: 'effort:one', path: 'reference.png', content: '' })

    render(<TeamFilesTab teamId="team-a" showEffortWorkspaces />)

    fireEvent.click(await screen.findByRole('button', { name: 'reference.png' }))
    expect(await screen.findByText('Preview unavailable from this transport')).toBeInTheDocument()
    expect(screen.getAllByText(/Image/).length).toBeGreaterThanOrEqual(2)
  })

  it('renders HTML in an isolated preview and keeps a source escape hatch', async () => {
    listTeamSharedFiles.mockResolvedValue([])
    listEffortWorkspaces.mockResolvedValue({
      teamId: 'team-a',
      workspaces: [{ effortRef: 'effort:one', slug: 'one', stage: 'review', files: [{ path: 'brief.html', isDir: false, size: 42 }] }],
      unavailable: [],
    })
    getEffortWorkspaceContent.mockResolvedValue({ effortRef: 'effort:one', path: 'brief.html', content: '<!doctype html><html><body><script>window.parent.postMessage("unsafe", "*")</script><h1>Brief</h1></body></html>' })

    render(<TeamFilesTab teamId="team-a" showEffortWorkspaces />)

    fireEvent.click(await screen.findByRole('button', { name: 'brief.html' }))
    const preview = await screen.findByTitle('brief.html rendered preview')
    expect(preview).toHaveAttribute('sandbox', '')
    const collectionPage = screen.getByTestId('team-file-artifact-preview').closest('[data-rcl-collection-page]')
    expect(collectionPage).toHaveAttribute('data-mobile-pane', 'inspector')
    const backToFiles = collectionPage?.querySelector<HTMLButtonElement>('[data-rcl-collection-back]')
    expect(backToFiles).toHaveTextContent('Back to files')
    fireEvent.click(backToFiles as HTMLButtonElement)
    expect(collectionPage).toHaveAttribute('data-mobile-pane', 'collection')
    const artifact = screen.getByTestId('team-file-artifact-preview')
    expect(within(artifact).getByRole('button', { name: 'Source' })).toBeInTheDocument()
    expect(within(artifact).getByRole('button', { name: 'Copy file path' })).toHaveAttribute('data-rcl-copy-button', '')
    fireEvent.click(within(artifact).getByRole('button', { name: 'Source' }))
    await waitFor(() => expect(artifact).toHaveAttribute('data-preview-mode', 'source'))
    expect(within(artifact).getByTestId('team-file-source')).toHaveTextContent('<!doctype html>')
    expect(within(artifact).getByRole('button', { name: 'Wrap' })).toBeInTheDocument()
  })
})

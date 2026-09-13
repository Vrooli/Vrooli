import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { TeamFilesTab } from './TeamFilesTab'
import * as teamService from '@/services/teamService'

vi.mock('@/services/teamService', async () => {
  const actual = await vi.importActual<typeof import('@/services/teamService')>('@/services/teamService')
  return {
    ...actual,
    listTeamSharedFiles: vi.fn(),
    getTeamSharedFileContent: vi.fn(),
    setTeamSharedFileContent: vi.fn(),
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
})

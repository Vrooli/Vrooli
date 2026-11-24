import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test-utils/renderWithProviders'
import { LineagePanel } from './LineagePanel'

vi.mock('./tabs/VersionHistoryTab', () => ({
  VersionHistoryTab: (props: { skillId: string }) => <div data-testid="mock-history">history:{props.skillId}</div>,
}))

describe('LineagePanel', () => {
  it('shows the version history for the skill', () => {
    render(<LineagePanel skillId="s42" />)
    expect(screen.getByTestId('mock-history')).toHaveTextContent('history:s42')
  })
})

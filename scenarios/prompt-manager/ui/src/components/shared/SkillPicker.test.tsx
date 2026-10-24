import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SkillPicker } from './SkillPicker'
import type { Skill } from '@/types'

function skill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'sk-1',
    name: 'Test Skill',
    description: 'A test skill',
    content: '',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    file: 'test.md',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('SkillPicker', () => {
  const defaultProps = {
    skills: [
      skill({ id: '1', name: 'Alpha Skill', description: 'First skill', tags: ['api'] }),
      skill({ id: '2', name: 'Beta Skill', description: 'Second skill', tags: ['docs'] }),
      skill({ id: '3', name: 'Gamma Skill', description: 'Third skill', tags: ['api', 'docs'] }),
    ],
    selectedIds: new Set<string>(),
    onToggle: vi.fn(),
  }

  it('renders search input', () => {
    render(<SkillPicker {...defaultProps} />)
    expect(screen.getByPlaceholderText('Search skills...')).toBeInTheDocument()
  })

  it('renders all skills initially', () => {
    render(<SkillPicker {...defaultProps} />)
    expect(screen.getByText('Alpha Skill')).toBeInTheDocument()
    expect(screen.getByText('Beta Skill')).toBeInTheDocument()
    expect(screen.getByText('Gamma Skill')).toBeInTheDocument()
  })

  it('filters skills by search query', () => {
    render(<SkillPicker {...defaultProps} />)

    fireEvent.change(screen.getByPlaceholderText('Search skills...'), {
      target: { value: 'Alpha' },
    })

    expect(screen.getByText('Alpha Skill')).toBeInTheDocument()
    expect(screen.queryByText('Beta Skill')).not.toBeInTheDocument()
    expect(screen.queryByText('Gamma Skill')).not.toBeInTheDocument()
  })

  it('filters by description text', () => {
    render(<SkillPicker {...defaultProps} />)

    fireEvent.change(screen.getByPlaceholderText('Search skills...'), {
      target: { value: 'Second' },
    })

    expect(screen.queryByText('Alpha Skill')).not.toBeInTheDocument()
    expect(screen.getByText('Beta Skill')).toBeInTheDocument()
  })

  it('shows correct selected count in footer', () => {
    render(
      <SkillPicker
        {...defaultProps}
        selectedIds={new Set(['1', '3'])}
      />,
    )

    expect(screen.getByText('2 of 3 selected')).toBeInTheDocument()
  })

  it('shows zero selected count', () => {
    render(<SkillPicker {...defaultProps} />)
    expect(screen.getByText('0 of 3 selected')).toBeInTheDocument()
  })

  it('handles empty skills array', () => {
    render(<SkillPicker {...defaultProps} skills={[]} />)
    expect(screen.getByText('0 of 0 selected')).toBeInTheDocument()
    expect(screen.getByText('No skills match your filters')).toBeInTheDocument()
  })

  it('shows all selected count', () => {
    render(
      <SkillPicker
        {...defaultProps}
        selectedIds={new Set(['1', '2', '3'])}
      />,
    )

    expect(screen.getByText('3 of 3 selected')).toBeInTheDocument()
  })

  it('search is case insensitive', () => {
    render(<SkillPicker {...defaultProps} />)

    fireEvent.change(screen.getByPlaceholderText('Search skills...'), {
      target: { value: 'alpha' },
    })

    expect(screen.getByText('Alpha Skill')).toBeInTheDocument()
  })
})

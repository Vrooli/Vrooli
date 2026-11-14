import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SkillCardView } from './SkillCardView'
import type { Skill } from '@/types'

function skill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'sk-1',
    name: 'Test Skill',
    description: 'A test description for preview',
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

describe('SkillCardView', () => {
  const defaultProps = {
    selectedItemId: null,
    onSelectItem: vi.fn(),
    dirtyItemIds: new Set<string>(),
    detailMode: 'full' as const,
  }

  it('renders empty state when no skills', () => {
    render(<SkillCardView {...defaultProps} skills={[]} />)
    expect(screen.getByText('No skills match your filters')).toBeInTheDocument()
  })

  it('renders skills as cards in a grid', () => {
    const skills = [
      skill({ id: '1', name: 'Alpha' }),
      skill({ id: '2', name: 'Beta' }),
    ]
    render(<SkillCardView {...defaultProps} skills={skills} />)
    expect(screen.getAllByTestId('skill-card-item')).toHaveLength(2)
    expect(screen.getByTestId('skill-card-view')).toHaveClass('grid-cols-2')
  })

  it('shows description preview', () => {
    const skills = [skill({ id: '1', description: 'Preview text here' })]
    render(<SkillCardView {...defaultProps} skills={skills} />)
    expect(screen.getByText('Preview text here')).toBeInTheDocument()
  })

  it('highlights selected card', () => {
    const skills = [skill({ id: '1' })]
    render(<SkillCardView {...defaultProps} skills={skills} selectedItemId="1" />)
    expect(screen.getByTestId('skill-card-item')).toHaveAttribute('aria-selected', 'true')
  })

  it('calls onSelectItem on click', () => {
    const onSelect = vi.fn()
    const skills = [skill({ id: '1' })]
    render(<SkillCardView {...defaultProps} skills={skills} onSelectItem={onSelect} />)

    fireEvent.click(screen.getByTestId('skill-card-item'))
    expect(onSelect).toHaveBeenCalledWith('1')
  })

  it('shows tags with overflow', () => {
    const skills = [skill({ id: '1', tags: ['a', 'b', 'c', 'd'] })]
    render(<SkillCardView {...defaultProps} skills={skills} />)
    expect(screen.getByText('a')).toBeInTheDocument()
    expect(screen.getByText('b')).toBeInTheDocument()
    expect(screen.getByText('+2')).toBeInTheDocument()
  })

  it('shows dirty indicator', () => {
    const skills = [skill({ id: '1' })]
    render(<SkillCardView {...defaultProps} skills={skills} dirtyItemIds={new Set(['1'])} />)
    expect(document.querySelector('.bg-amber-500')).toBeInTheDocument()
  })
})

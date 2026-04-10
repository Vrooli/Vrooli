import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SkillListView } from './SkillListView'
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

describe('SkillListView', () => {
  const defaultProps = {
    selectedItemId: null,
    onSelectItem: vi.fn(),
    dirtyItemIds: new Set<string>(),
    detailMode: 'full' as const,
  }

  it('renders empty state when no skills', () => {
    render(<SkillListView {...defaultProps} skills={[]} />)
    expect(screen.getByText('No skills match your filters')).toBeInTheDocument()
  })

  it('renders skills as list items', () => {
    const skills = [
      skill({ id: '1', name: 'Alpha' }),
      skill({ id: '2', name: 'Beta' }),
    ]
    render(<SkillListView {...defaultProps} skills={skills} />)
    expect(screen.getAllByTestId('skill-list-item')).toHaveLength(2)
    expect(screen.getByText('Alpha')).toBeInTheDocument()
    expect(screen.getByText('Beta')).toBeInTheDocument()
  })

  it('highlights selected item', () => {
    const skills = [skill({ id: '1', name: 'A' })]
    render(<SkillListView {...defaultProps} skills={skills} selectedItemId="1" />)
    expect(screen.getByTestId('skill-list-item')).toHaveAttribute('aria-selected', 'true')
  })

  it('calls onSelectItem on click', () => {
    const onSelect = vi.fn()
    const skills = [skill({ id: '1', name: 'A' })]
    render(<SkillListView {...defaultProps} skills={skills} onSelectItem={onSelect} />)

    fireEvent.click(screen.getByTestId('skill-list-item'))
    expect(onSelect).toHaveBeenCalledWith('1')
  })

  it('shows dirty indicator', () => {
    const skills = [skill({ id: '1', name: 'A' })]
    render(<SkillListView {...defaultProps} skills={skills} dirtyItemIds={new Set(['1'])} />)
    // Dirty dot is a 2x2 amber circle
    const dots = document.querySelectorAll('.bg-amber-500')
    expect(dots.length).toBeGreaterThan(0)
  })

  it('shows tags with overflow count', () => {
    const skills = [skill({ id: '1', tags: ['a', 'b', 'c', 'd', 'e'] })]
    render(<SkillListView {...defaultProps} skills={skills} />)
    expect(screen.getByText('a')).toBeInTheDocument()
    expect(screen.getByText('+2')).toBeInTheDocument()
  })

  it('shows metadata badges', () => {
    const skills = [skill({ id: '1', usageCount: 42, folder: 'core' })]
    render(<SkillListView {...defaultProps} skills={skills} />)
    expect(screen.getByText('42 uses')).toBeInTheDocument()
    expect(screen.getByText('core')).toBeInTheDocument()
  })

  it('fires context menu', () => {
    const onCtx = vi.fn()
    const skills = [skill({ id: '1', name: 'A', modes: ['dev'], folder: 'local' })]
    render(<SkillListView {...defaultProps} skills={skills} onSkillContextMenu={onCtx} />)

    fireEvent.contextMenu(screen.getByTestId('skill-list-item'))
    expect(onCtx).toHaveBeenCalledWith('1', 'A', expect.any(Number), expect.any(Number))
  })
})

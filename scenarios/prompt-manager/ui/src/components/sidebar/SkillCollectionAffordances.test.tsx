import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import type { Skill } from '@/types'
import { SkillCardView } from './SkillCardView'
import { SkillListView } from './SkillListView'
import { skillActions } from './skill-actions'

const testSkill: Skill = {
  id: 'skill-1',
  name: 'Collection contract',
  description: 'Shared action test',
  content: '',
  modes: ['development'],
  tags: [],
  draft: true,
  folder: 'local',
  file: 'collection.md',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  usageCount: 0,
}

const actions = skillActions({
  onOpen: vi.fn(),
  onCopy: vi.fn(),
  onMoveToFolder: vi.fn(),
  onChangeStorage: vi.fn(),
  onCreateNewFolder: vi.fn(),
  availableModePaths: [['development']],
})

const baseProps = {
  selectedItemId: null,
  onSelectItem: vi.fn(),
  dirtyItemIds: new Set<string>(),
  detailMode: 'full' as const,
  actions,
}

describe('skill collection affordances', () => {
  it.each([
    ['list', () => <SkillListView {...baseProps} skills={[testSkill]} />],
    ['card', () => <SkillCardView {...baseProps} skills={[testSkill]} />],
  ])('uses the shared action declaration in the %s presentation', (_name, view) => {
    render(view())

    fireEvent.contextMenu(screen.getByText('Collection contract'))

    expect(screen.getByRole('menuitem', { name: 'Open Enter' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Copy skill' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Move to development' })).toBeInTheDocument()
  })
})

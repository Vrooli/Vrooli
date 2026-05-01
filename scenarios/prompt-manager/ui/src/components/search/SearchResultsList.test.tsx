/**
 * Tests for SearchResultsList component.
 *
 * Covers: rank labels, chars badge, source badge refinement,
 * over-budget opacity, and non-discover mode.
 */

import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SearchResultsList } from './SearchResultsList'
import type { DiscoverResult, AISearchResult } from '@/lib/schemas'

function makeDiscoverResult(overrides: Partial<DiscoverResult> = {}): DiscoverResult {
  return {
    id: 'skill-1',
    name: 'Test Skill',
    description: 'A test skill',
    tags: ['tag1'],
    modes: ['steer'],
    score: 0.85,
    scorePercent: 85,
    source: 'search',
    topicId: '',
    topicName: '',
    contentChars: 1500,
    ...overrides,
  }
}

function makeSkillResult(overrides: Partial<AISearchResult> = {}): AISearchResult {
  return {
    id: 'skill-1',
    name: 'Test Skill',
    description: 'A test skill',
    folder: 'core',
    tags: ['tag1'],
    modes: ['steer'],
    score: 0.85,
    scorePercent: 85,
    ...overrides,
  }
}

const noopToggle = vi.fn()
const noopNavigate = vi.fn()

describe('SearchResultsList', () => {
  describe('Rank labels in discover mode', () => {
    it('renders rank numbers for discover results', () => {
      const results = [
        makeDiscoverResult({ id: 'a', name: 'Skill A' }),
        makeDiscoverResult({ id: 'b', name: 'Skill B' }),
        makeDiscoverResult({ id: 'c', name: 'Skill C' }),
      ]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
          compact
        />
      )

      expect(screen.getByText('#1')).toBeDefined()
      expect(screen.getByText('#2')).toBeDefined()
      expect(screen.getByText('#3')).toBeDefined()
    })

    it('renders Action discover results with operational metadata', () => {
      const results = [
        makeDiscoverResult({
          id: 'team.decisions.list',
          name: 'List Team Decisions',
          type: 'action',
          status: 'active',
          owner: 'scenario:prompt-manager',
          contentChars: 0,
        }),
      ]
      const onNavigate = vi.fn()

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={onNavigate}
        />
      )

      expect(screen.getByText('Action')).toBeDefined()
      expect(screen.getByLabelText('Action result')).toBeDefined()
      expect(screen.getByText('active')).toBeDefined()
      expect(screen.getByText('scenario:prompt-manager')).toBeDefined()

      fireEvent.click(screen.getByText('List Team Decisions'))
      expect(onNavigate).toHaveBeenCalledWith('team.decisions.list', 'action')
    })

    it('does not render rank numbers in non-discover skill mode', () => {
      const results = [makeSkillResult({ id: 'a', name: 'Skill A' })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={false}
          skillResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      expect(screen.queryByText('#1')).toBeNull()
    })
  })

  describe('CharsBadge', () => {
    it('formats sub-1K values', () => {
      const results = [makeDiscoverResult({ contentChars: 500 })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      expect(screen.getByText('500 chars')).toBeDefined()
    })

    it('formats 1K+ values with K suffix', () => {
      const results = [makeDiscoverResult({ contentChars: 2500 })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      expect(screen.getByText('2.5K chars')).toBeDefined()
    })
  })

  describe('SourceBadge', () => {
    it('shows topic name with blue styling for topic-sourced results', () => {
      const results = [makeDiscoverResult({
        source: 'topic',
        topicId: 'coaching',
        topicName: 'Coaching',
        topicDepth: 0,
      })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      const badge = screen.getByText('via Coaching')
      expect(badge).toBeDefined()
      expect(badge.className).toContain('bg-blue-500/10')
      expect(badge.className).toContain('text-blue-400')
    })

    it('falls back to topicId when topicName is empty', () => {
      const results = [makeDiscoverResult({
        source: 'topic',
        topicId: 'coaching',
        topicName: '',
        topicDepth: 1,
      })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      expect(screen.getByText('via coaching')).toBeDefined()
    })

    it('shows "direct match" with gray styling for search-sourced results', () => {
      const results = [makeDiscoverResult({ source: 'search' })]

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
        />
      )

      const badge = screen.getByText('direct match')
      expect(badge).toBeDefined()
      expect(badge.className).toContain('bg-muted')
    })
  })

  describe('Over-budget items', () => {
    it('applies opacity to over-budget items', () => {
      const results = [
        makeDiscoverResult({ id: 'a', name: 'In Budget' }),
        makeDiscoverResult({ id: 'b', name: 'Over Budget' }),
      ]

      const { container } = render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={false}
          selectedIds={new Set()}
          onToggleSelection={noopToggle}
          onNavigate={noopNavigate}
          overBudgetIds={new Set(['b'])}
        />
      )

      const rows = container.querySelectorAll('li > div')
      expect(rows[0]?.className).not.toContain('opacity-50')
      expect(rows[1]?.className).toContain('opacity-50')
    })

    it('keeps selection and navigation as separate controls in select mode', () => {
      const results = [makeDiscoverResult({ id: 'a', name: 'Selectable Skill' })]
      const onToggle = vi.fn()
      const onNavigate = vi.fn()

      render(
        <SearchResultsList
          entityType="skills"
          discoverMode={true}
          discoverResults={results}
          isSelectMode={true}
          selectedIds={new Set()}
          onToggleSelection={onToggle}
          onNavigate={onNavigate}
        />
      )

      fireEvent.click(screen.getByRole('button', { name: /Selectable Skill/ }))
      expect(onToggle).toHaveBeenCalledWith('a', 1500)
      expect(onNavigate).not.toHaveBeenCalled()

      fireEvent.click(screen.getByRole('button', { name: 'Go to entity' }))
      expect(onNavigate).toHaveBeenCalledWith('a', undefined)
    })
  })
})

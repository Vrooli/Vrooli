/**
 * Tests for SearchResultsList component.
 *
 * Covers: rank labels, chars badge, source badge refinement,
 * over-budget opacity, and non-discover mode.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
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

      const buttons = container.querySelectorAll('button')
      // First button (In Budget) should not have opacity-50
      expect(buttons[0]!.className).not.toContain('opacity-50')
      // Second button (Over Budget) should have opacity-50
      expect(buttons[1]!.className).toContain('opacity-50')
    })
  })
})

/**
 * Tests for MemberInboxTab — Gmail-style inbox surface for a member's
 * unrouted entries.
 *
 * Covers:
 * - Renders empty state when no intake declared
 * - Renders unrouted entries when intake + entries are present
 * - Promote action calls service with derived destination topic
 * - Drop action calls service and removes entry
 * - Refresh re-fetches entries
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemberInboxTab } from './MemberInboxTab'
import type { TopicIntakeEntry, TopicOutputEntry } from '@/types/topicsGraph'
import type { KnowledgeEntry } from '@/services/heartbeatService'

vi.mock('@/services/memberFlowService', () => ({
  listInboxEntries: vi.fn(),
  promoteInboxEntry: vi.fn(),
  dropInboxEntry: vi.fn(),
}))

import * as memberFlowService from '@/services/memberFlowService'

const mockedList = memberFlowService.listInboxEntries as unknown as ReturnType<typeof vi.fn>
const mockedPromote = memberFlowService.promoteInboxEntry as unknown as ReturnType<typeof vi.fn>
const mockedDrop = memberFlowService.dropInboxEntry as unknown as ReturnType<typeof vi.fn>

const intake: TopicIntakeEntry[] = [
  {
    prefix: 'research-inbox/*',
    taxonomy: 'marketing-research',
    classifier_skill: 'marketing-signal-classifier',
    source_team: null,
  },
]

const output: TopicOutputEntry[] = [
  { prefix: 'audience-scan/*', destination_kind: 'knowledge', destination_team: null, schema: 'audience-scan' },
  { prefix: 'competitor-record/*', destination_kind: 'knowledge', destination_team: null, schema: 'competitor-observation' },
]

const sampleEntry: KnowledgeEntry = {
  id: 'k-001',
  at: new Date(Date.now() - 5 * 60_000).toISOString(),
  topic: 'research-inbox/audience/foo-pain',
  content: 'A founder mentioned struggling with onboarding fatigue when wiring up MCP servers.',
  source: 'https://example.com/post/123',
  caller: 'vision-walk',
  attribution: {
    kind: 'writer-skill',
    member_id: null,
    team_id: null,
    run_id: null,
    spawn_origin: 'vision-walk',
    source_skill_id: 'morning-vision-walk',
  },
}

function firstIntakePrefix(): string {
  const entry = intake[0]
  if (!entry) {
    throw new Error('test fixture must include an intake entry')
  }
  return entry.prefix
}

describe('MemberInboxTab', () => {
  beforeEach(() => {
    mockedList.mockReset()
    mockedPromote.mockReset()
    mockedDrop.mockReset()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('renders empty state when intake is empty', () => {
    render(<MemberInboxTab teamId="marketing-crew" intake={[]} output={[]} />)
    expect(screen.getByTestId('inbox-no-intake')).toBeInTheDocument()
    expect(mockedList).not.toHaveBeenCalled()
  })

  it('fetches and renders unrouted entries for each intake prefix', async () => {
    mockedList.mockResolvedValue([sampleEntry])

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(mockedList).toHaveBeenCalledWith('marketing-crew', 'research-inbox/', { limit: 100 })
    })

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-entry-${sampleEntry.id}`)).toBeInTheDocument()
    })

    expect(screen.getByTestId(`inbox-count-${firstIntakePrefix()}`)).toHaveTextContent('1 unrouted')
    expect(screen.getByText('research-inbox/audience/foo-pain')).toBeInTheDocument()
    expect(screen.getByText(/by vision-walk/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /source/i })).toHaveAttribute('href', sampleEntry.source)
  })

  it('renders empty inbox message when prefix returns no entries', async () => {
    mockedList.mockResolvedValue([])

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-empty-${firstIntakePrefix()}`)).toBeInTheDocument()
    })
  })

  it('promote action retags entry to derived destination topic', async () => {
    mockedList.mockResolvedValue([sampleEntry])
    mockedPromote.mockResolvedValue({ ...sampleEntry, topic: 'audience-scan/foo-pain' })

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-promote-${sampleEntry.id}`)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByTestId(`inbox-promote-${sampleEntry.id}`))

    await waitFor(() => {
      // Default destination is the first knowledge output (audience-scan/*).
      // Slug is the last segment of the entry's topic ("foo-pain").
      expect(mockedPromote).toHaveBeenCalledWith(
        'marketing-crew',
        sampleEntry.id,
        'audience-scan/foo-pain',
      )
    })

    // After successful promote, the entry leaves the unrouted list.
    await waitFor(() => {
      expect(screen.queryByTestId(`inbox-entry-${sampleEntry.id}`)).not.toBeInTheDocument()
    })
  })

  it('promote uses the user-selected destination prefix', async () => {
    mockedList.mockResolvedValue([sampleEntry])
    mockedPromote.mockResolvedValue({ ...sampleEntry, topic: 'competitor-record/foo-pain' })

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-destination-${sampleEntry.id}`)).toBeInTheDocument()
    })

    fireEvent.change(screen.getByTestId(`inbox-destination-${sampleEntry.id}`), {
      target: { value: 'competitor-record/*' },
    })

    fireEvent.click(screen.getByTestId(`inbox-promote-${sampleEntry.id}`))

    await waitFor(() => {
      expect(mockedPromote).toHaveBeenCalledWith(
        'marketing-crew',
        sampleEntry.id,
        'competitor-record/foo-pain',
      )
    })
  })

  it('drop action deletes entry and removes it from the view', async () => {
    mockedList.mockResolvedValue([sampleEntry])
    mockedDrop.mockResolvedValue(undefined)

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-drop-${sampleEntry.id}`)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByTestId(`inbox-drop-${sampleEntry.id}`))

    await waitFor(() => {
      expect(mockedDrop).toHaveBeenCalledWith('marketing-crew', sampleEntry.id)
    })

    await waitFor(() => {
      expect(screen.queryByTestId(`inbox-entry-${sampleEntry.id}`)).not.toBeInTheDocument()
    })
  })

  it('refresh button re-fetches the entry list', async () => {
    mockedList.mockResolvedValueOnce([sampleEntry])

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(mockedList).toHaveBeenCalledTimes(1)
    })

    mockedList.mockResolvedValueOnce([])

    fireEvent.click(screen.getByLabelText(/refresh inbox/i))

    await waitFor(() => {
      expect(mockedList).toHaveBeenCalledTimes(2)
    })

    await waitFor(() => {
      expect(screen.getByTestId(`inbox-empty-${firstIntakePrefix()}`)).toBeInTheDocument()
    })
  })

  it('shows fallback message when output declares no knowledge destinations', async () => {
    mockedList.mockResolvedValue([sampleEntry])

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={[]} />)

    await waitFor(() => {
      expect(
        screen.getByText(/no knowledge destinations declared/i),
      ).toBeInTheDocument()
    })
    expect(screen.queryByTestId(`inbox-promote-${sampleEntry.id}`)).not.toBeInTheDocument()
    // Drop is still available.
    expect(screen.getByTestId(`inbox-drop-${sampleEntry.id}`)).toBeInTheDocument()
  })

  it('renders error message when listInboxEntries rejects', async () => {
    mockedList.mockRejectedValue(new Error('boom'))

    render(<MemberInboxTab teamId="marketing-crew" intake={intake} output={output} />)

    await waitFor(() => {
      expect(screen.getByText(/boom/)).toBeInTheDocument()
    })
  })
})

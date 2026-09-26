import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { MemberChatContextSelector } from './MemberChatContextSelector'

const agents = [
  { id: 'ada', name: 'Ada' },
  { id: 'bob', name: 'Bob' },
]

const memberships = [
  { teamId: 'team-a', teamName: 'Marketing', agentId: 'ada' },
  { teamId: 'team-b', teamName: 'Research', agentId: 'ada' },
  { teamId: 'team-c', teamName: 'Ops', agentId: 'bob' },
]

describe('MemberChatContextSelector', () => {
  it('offers the base context and only the selected agent memberships', () => {
    render(
      <MemberChatContextSelector
        agents={agents}
        memberships={memberships}
        value={{ agentId: 'ada' }}
        onChange={vi.fn()}
      />,
    )

    expect(screen.getByRole('combobox', { name: 'Agent' })).toHaveValue('ada')
    expect(screen.getByRole('option', { name: 'Base agent' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Marketing' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Research' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Ops' })).not.toBeInTheDocument()
  })

  it('emits the selected team-member identity', () => {
    const onChange = vi.fn()
    render(
      <MemberChatContextSelector
        agents={agents}
        memberships={memberships}
        value={{ agentId: 'ada' }}
        onChange={onChange}
      />,
    )

    fireEvent.change(screen.getByRole('combobox', { name: 'Context' }), {
      target: { value: 'team-b' },
    })

    expect(onChange).toHaveBeenCalledWith({ agentId: 'ada', teamId: 'team-b' })
  })

  it('returns to the base context when the agent changes', () => {
    const onChange = vi.fn()
    render(
      <MemberChatContextSelector
        agents={agents}
        memberships={memberships}
        value={{ agentId: 'ada', teamId: 'team-a' }}
        onChange={onChange}
      />,
    )

    fireEvent.change(screen.getByRole('combobox', { name: 'Agent' }), {
      target: { value: 'bob' },
    })

    expect(onChange).toHaveBeenCalledWith({ agentId: 'bob', teamId: undefined })
  })

  it('disables context selection until an agent is chosen', () => {
    render(
      <MemberChatContextSelector
        agents={agents}
        memberships={memberships}
        value={{ agentId: '' }}
        onChange={vi.fn()}
      />,
    )

    expect(screen.getByRole('combobox', { name: 'Agent' })).toHaveValue('')
    expect(screen.getByRole('combobox', { name: 'Context' })).toBeDisabled()
  })

  it('honours the disabled prop', () => {
    render(
      <MemberChatContextSelector
        agents={agents}
        memberships={memberships}
        value={{ agentId: 'ada' }}
        onChange={vi.fn()}
        disabled
      />,
    )

    expect(screen.getByRole('combobox', { name: 'Agent' })).toBeDisabled()
    expect(screen.getByRole('combobox', { name: 'Context' })).toBeDisabled()
  })
})

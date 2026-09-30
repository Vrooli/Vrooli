import { act, fireEvent, render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ConversationPanel } from './ConversationPanel'
import { conversationKey, createWorldConversations, hasUnread } from '../data/conversations'

beforeEach(() => { localStorage.clear(); Element.prototype.scrollIntoView = vi.fn(); document.exitPointerLock = vi.fn() })
describe('member conversation panel', () => {
  it('greets, sends to the member, displays a reply, and contains keyboard input', async () => {
    const member = { id: 'ada', name: 'Ada', teamId: 'garden' }
    const run = { id: 'run-1', taskId: 'task-1', status: 'running' }
    const api = { start: vi.fn().mockResolvedValue(run), continue: vi.fn(), details: vi.fn().mockResolvedValue(run), events: vi.fn().mockResolvedValue([{ id: 'reply', sequence: 2, eventType: 'message', data: { role: 'assistant', content: 'Hello from the garden.' } }]) }
    const conversations = createWorldConversations(api)
    const close = vi.fn(), movement = vi.fn()
    render(<div onKeyDown={movement}><ConversationPanel member={member} conversations={conversations} preview={false} onClose={close} /></div>)
    expect(screen.getByText('Hi, I’m Ada. What would you like to work on?')).toBeVisible()
    const input = screen.getByRole('textbox', { name: 'Message Ada' })
    fireEvent.focus(input); expect(document.exitPointerLock).toHaveBeenCalled()
    fireEvent.change(input, { target: { value: 'How are you?' } })
    fireEvent.keyDown(input, { key: 'w' }); expect(movement).not.toHaveBeenCalled()
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(api.start).toHaveBeenCalledWith(expect.objectContaining({ agentId: 'ada', teamId: 'garden', message: 'How are you?' })))
    await act(() => conversations.refresh(conversationKey(member)))
    expect(screen.getByText('Hello from the garden.')).toBeVisible()
    expect(hasUnread(conversations.getSnapshot()[conversationKey(member)])).toBe(false)
    expect(screen.getByRole('link', { name: 'Open full conversation' })).toHaveAttribute('href', '/runs/run-1')
    fireEvent.keyDown(input, { key: 'Escape' }); expect(close).toHaveBeenCalled()
  })
  it('never offers a real send for synthetic preview agents', () => {
    render(<ConversationPanel member={{ id: 'preview', name: 'Preview' }} conversations={createWorldConversations()} preview onClose={vi.fn()} />)
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.getByText(/Preview agent/)).toBeVisible()
  })
})

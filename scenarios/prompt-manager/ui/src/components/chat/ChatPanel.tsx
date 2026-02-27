/**
 * ChatPanel - Simplified chat interface for agent conversations.
 *
 * Displays messages extracted from run events and provides input
 * for continuing the conversation.
 */

import { useState, useRef, useEffect, useMemo } from 'react'
import { Send, Loader2, User, Bot } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { RunDetails, RunEvent } from '@/services/heartbeatService'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}

interface ChatPanelProps {
  run: RunDetails | null
  events: RunEvent[]
  eventsLoading: boolean
  onContinue: (message: string) => Promise<void>
}

export function ChatPanel({ run, events, eventsLoading, onContinue }: ChatPanelProps) {
  const [input, setInput] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const messages = useMemo<ChatMessage[]>(() => {
    return events
      .filter((e) => e.eventType === 'message')
      .map((e) => {
        const role = String(e.data.role ?? '').toLowerCase()
        if (role !== 'user' && role !== 'assistant') return null
        return {
          id: e.id,
          role: role as 'user' | 'assistant',
          content: String(e.data.content ?? ''),
          timestamp: new Date(e.timestamp),
        }
      })
      .filter((m): m is ChatMessage => m !== null)
  }, [events])

  const isGenerating = run != null && ['running', 'pending'].includes(run.status)
  const canContinue = run?.actions?.canContinue === true

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length, isGenerating])

  // Auto-resize textarea
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
      textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 200)}px`
    }
  }, [input])

  const handleSend = async () => {
    const trimmed = input.trim()
    if (!trimmed || isSending) return
    setIsSending(true)
    setError(null)
    try {
      await onContinue(trimmed)
      setInput('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message')
    } finally {
      setIsSending(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {eventsLoading && messages.length === 0 && (
          <div className="flex items-center justify-center py-8 text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin mr-2" />
            Loading messages...
          </div>
        )}

        {messages.map((msg) => (
          <div
            key={msg.id}
            className={cn(
              'flex gap-3',
              msg.role === 'user' ? 'flex-row-reverse' : 'flex-row',
            )}
          >
            {/* Avatar */}
            <div
              className={cn(
                'h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0',
                msg.role === 'user' ? 'bg-primary/20 text-primary' : 'bg-muted text-muted-foreground',
              )}
            >
              {msg.role === 'user' ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
            </div>

            {/* Bubble */}
            <div
              className={cn(
                'max-w-[80%] rounded-lg px-4 py-2',
                msg.role === 'user'
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-foreground',
              )}
            >
              <p className="text-sm whitespace-pre-wrap break-words">{msg.content}</p>
              <p className="text-xs opacity-60 mt-1">
                {msg.timestamp.toLocaleTimeString()}
              </p>
            </div>
          </div>
        ))}

        {/* Generating skeleton */}
        {isGenerating && (
          <div className="flex gap-3">
            <div className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 bg-muted text-muted-foreground">
              <Bot className="h-4 w-4" />
            </div>
            <div className="bg-muted rounded-lg px-4 py-3">
              <p className="text-xs text-muted-foreground mb-2">Generating response...</p>
              <div className="space-y-2">
                <div className="h-3 w-[200px] bg-foreground/10 rounded animate-pulse" />
                <div className="h-3 w-[160px] bg-foreground/10 rounded animate-pulse" />
                <div className="h-3 w-[180px] bg-foreground/10 rounded animate-pulse" />
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Error display */}
      {error && (
        <div className="mx-4 mb-2 px-3 py-2 bg-destructive/20 text-destructive text-sm rounded-lg">
          {error}
        </div>
      )}

      {/* Input area */}
      {canContinue && (
        <div className="border-t border-border p-4">
          <div className="flex gap-2">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Type a message..."
              disabled={isSending}
              rows={1}
              className={cn(
                'flex-1 resize-none rounded-lg border border-border bg-background px-3 py-2 text-sm',
                'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring',
                isSending && 'opacity-50',
              )}
            />
            <button
              type="button"
              onClick={handleSend}
              disabled={isSending || !input.trim()}
              className={cn(
                'h-10 w-10 flex items-center justify-center rounded-lg',
                'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
                (isSending || !input.trim()) && 'opacity-50 cursor-not-allowed',
              )}
              aria-label="Send message"
            >
              {isSending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            </button>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Press Enter to send, Shift+Enter for new line
          </p>
        </div>
      )}

      {/* Terminal state message */}
      {run && !canContinue && !isGenerating && (
        <div className="border-t border-border p-4 text-center text-sm text-muted-foreground">
          {run.status === 'completed' ? 'Chat session completed.' : `Run status: ${run.status}`}
        </div>
      )}
    </div>
  )
}

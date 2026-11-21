import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import { conversationKey, type ConversationMember, type WorldConversations } from '../data/conversations'

export function ConversationPanel({ member, conversations, preview, onClose }: {
  member: ConversationMember; conversations: WorldConversations; preview: boolean; onClose: () => void
}) {
  const [input, setInput] = useState(() => { const saved = conversations.getSnapshot()[conversationKey(member)]; return saved && !saved.runId ? saved.firstMessage : '' })
  const [sendError, setSendError] = useState<string>()
  const snapshot = useSyncExternalStore(conversations.subscribe, conversations.getSnapshot)
  const key = conversationKey(member)
  const conversation = snapshot[key]
  const end = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const read = () => { if (document.visibilityState === 'visible') conversations.read(key) }
    read()
    document.addEventListener('visibilitychange', read)
    return () => document.removeEventListener('visibilitychange', read)
  }, [conversation?.after, key, conversations])
  useEffect(() => { end.current?.scrollIntoView({ block: 'nearest' }) }, [conversation?.messages.length])
  const canSend = !preview && !conversation?.sending && (!conversation?.runId || conversation.run?.actions?.canContinue === true)
  const send = async () => {
    if (!canSend) return
    setSendError(undefined)
    try { await conversations.send(member, input.trim()); setInput('') }
    catch (error) { setSendError(error instanceof Error ? error.message : 'Could not send message') }
  }
  return (
      <section aria-label={`Conversation with ${member.name}`} data-testid="world-conversation" className="w-[min(320px,85vw)] rounded-2xl border border-border bg-background text-foreground shadow-xl"
        onPointerDown={event => { event.stopPropagation(); document.exitPointerLock() }} onClick={event => event.stopPropagation()} onWheel={event => event.stopPropagation()}
        onKeyDown={event => { event.stopPropagation(); if (event.key === 'Escape') { event.preventDefault(); onClose() } }}>
        <header className="flex items-center justify-between border-b px-3 py-2"><strong className="truncate text-sm">{member.name}</strong><button type="button" aria-label="Close conversation" onClick={onClose} className="px-2 py-1">×</button></header>
        <div className="max-h-[min(32vh,240px)] overflow-y-auto space-y-2 p-3 text-sm" role="log" aria-live="polite">
          <p>Hi, I’m {member.name}. What would you like to work on?</p>
          {conversation?.firstMessage && <p className="rounded-lg bg-primary/15 p-2"><span className="sr-only">You: </span>{conversation.firstMessage}</p>}
          {conversation?.messages.filter(message => !(message.role === 'user' && message.content === conversation.firstMessage)).map(message => <p key={message.id} className={`whitespace-pre-wrap break-words rounded-lg p-2 ${message.role === 'user' ? 'bg-primary/15' : 'bg-muted'}`}><span className="sr-only">{message.role === 'user' ? 'You' : member.name}: </span>{message.content}</p>)}
          {conversation?.run && ['pending', 'starting', 'running'].includes(conversation.run.status) && <p role="status" className="text-muted-foreground">Thinking…</p>}
          <div ref={end} />
        </div>
        <div className="border-t p-3 space-y-2">
          {preview ? <p className="text-xs text-muted-foreground">Preview agent. Open the live world to send messages.</p> : <>
            {(sendError || conversation?.error) && <p role="alert" className="text-xs text-destructive">{sendError ?? conversation?.error}</p>}
            <textarea aria-label={`Message ${member.name}`} rows={2} value={input} onFocus={() => document.exitPointerLock()} onChange={event => setInput(event.target.value)}
              onKeyDown={event => { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void send() } }}
              placeholder="Type a message…" className="w-full resize-none rounded-lg border bg-background p-2 text-sm" />
            <div className="flex items-center justify-between gap-2">
              {conversation?.runId && <a href={`/runs/${encodeURIComponent(conversation.runId)}`} className="text-xs underline">Open full conversation</a>}
              <button type="button" onClick={() => void send()} disabled={!canSend || !input.trim()} className="ml-auto rounded-lg bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50">{conversation?.sending ? 'Sending…' : 'Send'}</button>
            </div>
            {conversation?.run && !canSend && !conversation.sending && !['pending', 'starting', 'running'].includes(conversation.run.status) && <p className="text-xs text-muted-foreground">This session cannot continue. Open the full conversation for its status and recovery actions.</p>}
          </>}
        </div>
      </section>
  )
}

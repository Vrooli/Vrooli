/**
 * StartChatDialog - Two-phase modal for launching agent chat from skill view.
 *
 * Phase 1 (Configure): User writes a message, selects skills for context.
 * Phase 2 (Active): Shows ChatPanel with live conversation.
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { Search, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Dialog } from '@/components/shared/Dialog'
import { ChatPanel } from './ChatPanel'
import { getSkill } from '@/services/skillService'
import {
  createTask,
  createRun,
  getRunDetails,
  getRunEvents,
  continueRun,
  type RunDetails,
  type RunEvent,
} from '@/services/heartbeatService'
import type { Skill } from '@/types'

interface StartChatDialogProps {
  isOpen: boolean
  onClose: () => void
  initialSkill: Skill
  allSkills: Skill[]
}

type Phase = 'configure' | 'active'

export function StartChatDialog({ isOpen, onClose, initialSkill, allSkills }: StartChatDialogProps) {
  const [phase, setPhase] = useState<Phase>('configure')
  const [message, setMessage] = useState('')
  const [selectedSkillIds, setSelectedSkillIds] = useState<Set<string>>(new Set([initialSkill.id]))
  const [skillFilter, setSkillFilter] = useState('')
  const [isLaunching, setIsLaunching] = useState(false)
  const [launchError, setLaunchError] = useState<string | null>(null)

  // Active phase state
  const [run, setRun] = useState<RunDetails | null>(null)
  const [events, setEvents] = useState<RunEvent[]>([])
  const maxSequenceRef = useRef(-1)
  const eventPollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const runPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Reset state when dialog opens
  useEffect(() => {
    if (isOpen) {
      setPhase('configure')
      setMessage('')
      setSelectedSkillIds(new Set([initialSkill.id]))
      setSkillFilter('')
      setIsLaunching(false)
      setLaunchError(null)
      setRun(null)
      setEvents([])
      maxSequenceRef.current = -1
    }
  }, [isOpen, initialSkill.id])

  // Clean up polling on unmount/close
  useEffect(() => {
    return () => {
      if (eventPollRef.current) clearInterval(eventPollRef.current)
      if (runPollRef.current) clearInterval(runPollRef.current)
    }
  }, [])

  const isTerminal = (status: string) =>
    ['completed', 'failed', 'cancelled', 'error'].includes(status)

  const startPolling = useCallback((runId: string) => {
    // Poll events every 2s
    const pollEvents = async () => {
      try {
        const newEvents = await getRunEvents(runId, {
          afterSequence: maxSequenceRef.current,
        })
        if (newEvents.length > 0) {
          setEvents((prev) => [...prev, ...newEvents])
          maxSequenceRef.current = Math.max(
            maxSequenceRef.current,
            ...newEvents.map((e) => e.sequence),
          )
        }
      } catch {
        // Silently handle polling errors
      }
    }

    // Poll run details every 5s
    const pollRun = async () => {
      try {
        const details = await getRunDetails(runId)
        setRun(details)
        if (isTerminal(details.status)) {
          if (eventPollRef.current) clearInterval(eventPollRef.current)
          if (runPollRef.current) clearInterval(runPollRef.current)
          // One final event fetch
          pollEvents()
        }
      } catch {
        // Silently handle polling errors
      }
    }

    // Initial fetches
    pollEvents()
    pollRun()

    eventPollRef.current = setInterval(pollEvents, 2000)
    runPollRef.current = setInterval(pollRun, 5000)
  }, [])

  const handleStartChat = async () => {
    const trimmed = message.trim()
    if (!trimmed) return

    setIsLaunching(true)
    setLaunchError(null)

    try {
      // 1. Fetch full content for selected skills
      const skillContents: string[] = []
      for (const skillId of selectedSkillIds) {
        const skill = await getSkill(skillId)
        if (skill?.content) {
          skillContents.push(`## Skill: ${skill.name}\n\n${skill.content}`)
        }
      }

      // 2. Build task description
      let description = trimmed
      if (skillContents.length > 0) {
        description += '\n\n---\n\n# Context (Skills)\n\n' + skillContents.join('\n\n---\n\n')
      }

      // 3. Create task
      const task = await createTask({
        title: trimmed.slice(0, 100) + (trimmed.length > 100 ? '...' : ''),
        description,
        scopePath: '.',
      })

      // 4. Create run
      const newRun = await createRun({ taskId: task.id })
      setRun(newRun)
      setPhase('active')

      // 5. Start polling
      startPolling(newRun.id)
    } catch (err) {
      setLaunchError(err instanceof Error ? err.message : 'Failed to start chat')
    } finally {
      setIsLaunching(false)
    }
  }

  const handleContinue = async (msg: string) => {
    if (!run) return
    await continueRun(run.id, msg)
  }

  const handleClose = () => {
    if (eventPollRef.current) clearInterval(eventPollRef.current)
    if (runPollRef.current) clearInterval(runPollRef.current)
    onClose()
  }

  const toggleSkill = (id: string) => {
    setSelectedSkillIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const filteredSkills = allSkills.filter((s) => {
    if (!skillFilter) return true
    const lower = skillFilter.toLowerCase()
    return s.name.toLowerCase().includes(lower) || (s.description ?? '').toLowerCase().includes(lower)
  })

  return (
    <Dialog
      isOpen={isOpen}
      onClose={handleClose}
      title={phase === 'configure' ? 'Start Agent Chat' : 'Agent Chat'}
      maxWidth="max-w-2xl"
      isLoading={isLaunching}
      className={phase === 'active' ? 'flex flex-col h-[80vh]' : undefined}
    >
      {phase === 'configure' ? (
        <div className="space-y-4">
          {/* Message input */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Message
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="What would you like to discuss?"
              rows={3}
              className={cn(
                'w-full rounded-lg border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white',
                'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary',
              )}
              autoFocus
            />
          </div>

          {/* Skill selector */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Skills for context ({selectedSkillIds.size} selected)
            </label>
            <div className="relative mb-2">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
              <input
                type="text"
                value={skillFilter}
                onChange={(e) => setSkillFilter(e.target.value)}
                placeholder="Filter skills..."
                className={cn(
                  'w-full rounded-lg border border-white/10 bg-slate-800 pl-9 pr-3 py-2 text-sm text-white',
                  'placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary',
                )}
              />
            </div>
            <div className="max-h-48 overflow-y-auto space-y-1 rounded-lg border border-white/10 bg-slate-800/50 p-2">
              {filteredSkills.map((skill) => (
                <label
                  key={skill.id}
                  className={cn(
                    'flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer hover:bg-white/5',
                    selectedSkillIds.has(skill.id) && 'bg-primary/10',
                  )}
                >
                  <input
                    type="checkbox"
                    checked={selectedSkillIds.has(skill.id)}
                    onChange={() => toggleSkill(skill.id)}
                    className="rounded border-white/20"
                  />
                  <span className="text-sm text-slate-200 truncate">{skill.name}</span>
                  {skill.id === initialSkill.id && (
                    <span className="text-xs text-primary ml-auto flex-shrink-0">current</span>
                  )}
                </label>
              ))}
              {filteredSkills.length === 0 && (
                <p className="text-sm text-slate-500 text-center py-2">No skills match filter</p>
              )}
            </div>
          </div>

          {/* Error */}
          {launchError && (
            <div className="px-3 py-2 bg-destructive/20 text-destructive text-sm rounded-lg">
              {launchError}
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 text-sm rounded-lg border border-white/10 text-slate-300 hover:bg-white/5 transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleStartChat}
              disabled={!message.trim() || isLaunching}
              className={cn(
                'px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors',
                'flex items-center gap-2',
                (!message.trim() || isLaunching) && 'opacity-50 cursor-not-allowed',
              )}
            >
              {isLaunching && <Loader2 className="h-4 w-4 animate-spin" />}
              Start Chat
            </button>
          </div>
        </div>
      ) : (
        <div className="flex-1 overflow-hidden -m-6">
          <ChatPanel
            run={run}
            events={events}
            eventsLoading={false}
            onContinue={handleContinue}
          />
        </div>
      )}
    </Dialog>
  )
}

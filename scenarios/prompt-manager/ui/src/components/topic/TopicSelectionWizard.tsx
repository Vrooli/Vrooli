/**
 * TopicSelectionWizard - Multi-step wizard for discovering skills through topic hierarchy.
 *
 * Steps:
 * 1. Root topics: Select top-level topics
 * 2. Drilldown: Narrow by selecting child topics (auto-skips single-child levels)
 * 3. Review: See accumulated skills, toggle individual ones, pick complexity
 *
 * The copy footer is always visible so users can skip ahead at any step.
 */

import { useState, useCallback, useMemo, useEffect } from 'react'
import { X, ChevronRight, ChevronLeft, Check, Copy, FileCode, FileText, Braces, Terminal } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTopics } from '@/hooks/useTopicData'
import { getAccumulatedSkills } from '@/services/topicService'
import { api } from '@/lib/api'
import { copyToClipboard } from '@/lib/clipboard'
import { toast } from '@/hooks/use-toast'
import type { Topic } from '@/lib/schemas/topic.schema'
import type { CombineFormat } from '@/stores/combineStore'
import { getSavedFormat, saveFormat } from '@/lib/formatPreference'

interface TopicSelectionWizardProps {
  onClose: () => void
  className?: string
}

type WizardStep = 'roots' | 'drilldown' | 'review'

type Complexity = 'simple' | 'standard' | 'detailed'

const FORMAT_OPTIONS: Array<{ value: CombineFormat; label: string; icon: React.ReactNode }> = [
  { value: 'xml', label: 'XML', icon: <FileCode className="h-3.5 w-3.5" /> },
  { value: 'markdown', label: 'MD', icon: <FileText className="h-3.5 w-3.5" /> },
  { value: 'json', label: 'JSON', icon: <Braces className="h-3.5 w-3.5" /> },
  { value: 'cli', label: 'CLI', icon: <Terminal className="h-3.5 w-3.5" /> },
]

const COMPLEXITY_OPTIONS: Array<{ value: Complexity; label: string }> = [
  { value: 'simple', label: 'Simple' },
  { value: 'standard', label: 'Standard' },
  { value: 'detailed', label: 'Detailed' },
]

const MAX_DRILLDOWN_DEPTH = 3

export function TopicSelectionWizard({ onClose, className }: TopicSelectionWizardProps) {
  const { topics } = useTopics()

  // Wizard state
  const [step, setStep] = useState<WizardStep>('roots')
  const [selectedTopicIds, setSelectedTopicIds] = useState<Set<string>>(new Set())
  const [deselectedSkillIds, setDeselectedSkillIds] = useState<Set<string>>(new Set())
  const [complexity, setComplexity] = useState<Complexity>('standard')
  const [format, setFormatState] = useState<CombineFormat>(getSavedFormat)
  const setFormat = useCallback((f: CombineFormat) => {
    saveFormat(f)
    setFormatState(f)
  }, [])

  // Drilldown state
  const [drilldownBreadcrumbs, setDrilldownBreadcrumbs] = useState<Array<{ id: string; name: string }>>([])
  const [drilldownParentId, setDrilldownParentId] = useState<string | null>(null)
  const [drilldownQueue, setDrilldownQueue] = useState<string[]>([])

  // Review state
  const [accumulatedSkillIds, setAccumulatedSkillIds] = useState<string[]>([])
  const [isLoadingSkills, setIsLoadingSkills] = useState(false)

  // Copy state
  const [isCopying, setIsCopying] = useState(false)
  const [copySuccess, setCopySuccess] = useState(false)

  // Derived data
  const rootTopics = useMemo(
    () => topics.filter((t) => !t.parentTopicId),
    [topics],
  )

  const childrenOf = useCallback(
    (parentId: string) => topics.filter((t) => t.parentTopicId === parentId),
    [topics],
  )

  const currentChildren = useMemo(
    () => (drilldownParentId ? childrenOf(drilldownParentId) : []),
    [drilldownParentId, childrenOf],
  )

  // Count how many skills are effectively selected (accumulated minus deselected)
  const effectiveSkillCount = accumulatedSkillIds.length - deselectedSkillIds.size

  // Selection helpers
  const toggleTopic = useCallback((id: string) => {
    setSelectedTopicIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const toggleAllTopics = useCallback((topicIds: string[], selected: boolean) => {
    setSelectedTopicIds((prev) => {
      const next = new Set(prev)
      for (const id of topicIds) {
        if (selected) {
          next.add(id)
        } else {
          next.delete(id)
        }
      }
      return next
    })
  }, [])

  const toggleSkill = useCallback((skillId: string) => {
    setDeselectedSkillIds((prev) => {
      const next = new Set(prev)
      if (next.has(skillId)) {
        next.delete(skillId)
      } else {
        next.add(skillId)
      }
      return next
    })
  }, [])

  // Get the selected skill IDs (for copy operations at any step)
  const getSelectedSkillIds = useCallback((): string[] => {
    if (step === 'review') {
      return accumulatedSkillIds.filter((id) => !deselectedSkillIds.has(id))
    }
    // For roots/drilldown, collect skills from all selected topics
    const skillIds = new Set<string>()
    for (const topicId of selectedTopicIds) {
      const topic = topics.find((t) => t.id === topicId)
      if (topic?.skills) {
        for (const s of topic.skills) skillIds.add(s)
      }
    }
    return Array.from(skillIds)
  }, [step, accumulatedSkillIds, deselectedSkillIds, selectedTopicIds, topics])

  // Navigation: roots -> drilldown -> review
  const handleNext = useCallback(() => {
    if (step === 'roots') {
      // Find selected root topics that have children
      const parentsWithChildren = Array.from(selectedTopicIds).filter(
        (id) => childrenOf(id).length > 0,
      )

      if (parentsWithChildren.length === 0) {
        // No children to drill down into, skip to review
        setStep('review')
        return
      }

      // Start drilldown with the first parent
      const firstParent = parentsWithChildren[0]
      if (firstParent === undefined) return
      const firstTopic = topics.find((t) => t.id === firstParent)
      const children = childrenOf(firstParent)

      // Auto-select and skip if only one child
      const onlyChild = children.length === 1 ? children[0] : undefined
      if (onlyChild) {
        setSelectedTopicIds((prev) => new Set([...prev, onlyChild.id]))
        // Continue to next parent or review
        const remaining = parentsWithChildren.slice(1)
        if (remaining.length === 0) {
          setStep('review')
          return
        }
        setDrilldownQueue(remaining.slice(1))
        const next = remaining[0]
        if (next === undefined) return
        const nextTopic = topics.find((t) => t.id === next)
        setDrilldownParentId(next)
        setDrilldownBreadcrumbs([{ id: next, name: nextTopic?.name ?? next }])
        setStep('drilldown')
        return
      }

      setDrilldownParentId(firstParent)
      setDrilldownBreadcrumbs([{ id: firstParent, name: firstTopic?.name ?? firstParent }])
      setDrilldownQueue(parentsWithChildren.slice(1))
      setStep('drilldown')
    } else if (step === 'drilldown') {
      // Move to next parent in queue or go to review
      if (drilldownQueue.length > 0) {
        const nextParent = drilldownQueue[0]
        if (nextParent === undefined) return
        const nextTopic = topics.find((t) => t.id === nextParent)
        const children = childrenOf(nextParent)

        // Auto-select and skip single child
        const singleChild = children.length === 1 ? children[0] : undefined
        if (singleChild) {
          setSelectedTopicIds((prev) => new Set([...prev, singleChild.id]))
          setDrilldownQueue((prev) => prev.slice(1))
          if (drilldownQueue.length <= 1) {
            setStep('review')
          }
          return
        }

        setDrilldownParentId(nextParent)
        setDrilldownBreadcrumbs([{ id: nextParent, name: nextTopic?.name ?? nextParent }])
        setDrilldownQueue((prev) => prev.slice(1))
      } else {
        setStep('review')
      }
    }
  }, [step, selectedTopicIds, topics, childrenOf, drilldownQueue])

  const handleBack = useCallback(() => {
    if (step === 'drilldown') {
      if (drilldownBreadcrumbs.length > 1) {
        // Go back one level in the breadcrumb
        const newBreadcrumbs = drilldownBreadcrumbs.slice(0, -1)
        const parent = newBreadcrumbs[newBreadcrumbs.length - 1]
        if (parent === undefined) return
        setDrilldownBreadcrumbs(newBreadcrumbs)
        setDrilldownParentId(parent.id)
      } else {
        setStep('roots')
        setDrilldownBreadcrumbs([])
        setDrilldownParentId(null)
      }
    } else if (step === 'review') {
      setStep('roots')
      setDrilldownBreadcrumbs([])
      setDrilldownParentId(null)
      setDrilldownQueue([])
    }
  }, [step, drilldownBreadcrumbs])

  // Drilldown into a child topic's children
  const handleDrillInto = useCallback(
    (topicId: string) => {
      const children = childrenOf(topicId)
      if (children.length === 0) return

      const topic = topics.find((t) => t.id === topicId)
      const depth = drilldownBreadcrumbs.length + 1

      if (depth >= MAX_DRILLDOWN_DEPTH) {
        // Flatten: auto-select all descendants
        const descendants = getAllDescendants(topicId, topics)
        setSelectedTopicIds((prev) => {
          const next = new Set(prev)
          for (const d of descendants) next.add(d)
          return next
        })
        return
      }

      // Auto-select and skip single child
      const onlyChild = children.length === 1 ? children[0] : undefined
      if (onlyChild) {
        setSelectedTopicIds((prev) => new Set([...prev, onlyChild.id]))
        return
      }

      setDrilldownBreadcrumbs((prev) => [...prev, { id: topicId, name: topic?.name ?? topicId }])
      setDrilldownParentId(topicId)
    },
    [childrenOf, topics, drilldownBreadcrumbs],
  )

  // Load accumulated skills when entering review step
  useEffect(() => {
    if (step !== 'review') return

    let cancelled = false
    setIsLoadingSkills(true)
    setDeselectedSkillIds(new Set())

    const selectedIds = Array.from(selectedTopicIds)

    void Promise.all(
      selectedIds.map((id) =>
        getAccumulatedSkills(id).catch(() => ({ topicId: id, ancestry: [], skills: [] })),
      ),
    ).then((results) => {
      if (cancelled) return
      const allSkills = new Set<string>()
      for (const r of results) {
        for (const s of r.skills) allSkills.add(s)
      }
      setAccumulatedSkillIds(Array.from(allSkills))
      setIsLoadingSkills(false)
    })

    return () => {
      cancelled = true
    }
  }, [step, selectedTopicIds])

  // Pre-fetch combined content so the copy click is synchronous (preserves
  // user activation on iOS).
  const [prefetchedContent, setPrefetchedContent] = useState<string | null>(null)
  const selectedSkillIds = getSelectedSkillIds()

  useEffect(() => {
    setPrefetchedContent(null)
    if (selectedSkillIds.length === 0) return

    let stale = false
    void api.displaySkills(selectedSkillIds, format)
      .then((r) => { if (!stale) setPrefetchedContent(r.combined) })
      .catch(() => { /* prefetch failed — handleCopy will show error */ })
    return () => { stale = true }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- selectedSkillIds is a new array each render; stringify for stable dep
  }, [JSON.stringify(selectedSkillIds), format])

  // Copy handler (works on any step)
  const handleCopy = useCallback(() => {
    const skillIds = getSelectedSkillIds()
    if (skillIds.length === 0) return

    if (prefetchedContent === null) {
      toast({
        title: 'Still loading',
        description: 'Content is being prepared — please try again in a moment.',
      })
      return
    }

    setIsCopying(true)
    setCopySuccess(false)

    copyToClipboard(prefetchedContent)
      .then(() => {
        setCopySuccess(true)
        toast({
          title: 'Copied to clipboard',
          description: `${skillIds.length} skill${skillIds.length !== 1 ? 's' : ''} as ${format.toUpperCase()}`,
        })
        setTimeout(() => setCopySuccess(false), 2000)
      })
      .catch((error: unknown) => {
        const msg = error instanceof Error ? error.message : String(error)
        console.error('Failed to copy skills:', error)
        toast({
          title: 'Copy failed',
          description: msg,
          variant: 'destructive',
        })
      })
      .finally(() => {
        setIsCopying(false)
      })
  }, [getSelectedSkillIds, format, prefetchedContent])

  const copyableCount = step === 'review' ? effectiveSkillCount : getSelectedSkillIds().length

  const stepNumber = step === 'roots' ? 1 : step === 'drilldown' ? 2 : 3

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <div className="flex items-center gap-3">
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-muted"
            title="Close wizard"
          >
            <X className="w-4 h-4" />
          </button>
          <h2 className="text-sm font-semibold">Discover Skills</h2>
        </div>
        <span className="text-xs text-muted-foreground">
          Step {stepNumber} of 3
        </span>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        {step === 'roots' && (
          <RootsStep
            topics={rootTopics}
            selectedIds={selectedTopicIds}
            childrenOf={childrenOf}
            onToggle={toggleTopic}
            onToggleAll={toggleAllTopics}
          />
        )}

        {step === 'drilldown' && (
          <DrilldownStep
            children={currentChildren}
            selectedIds={selectedTopicIds}
            breadcrumbs={drilldownBreadcrumbs}
            childrenOf={childrenOf}
            onToggle={toggleTopic}
            onToggleAll={toggleAllTopics}
            onDrillInto={handleDrillInto}
          />
        )}

        {step === 'review' && (
          <ReviewStep
            skillIds={accumulatedSkillIds}
            deselectedIds={deselectedSkillIds}
            isLoading={isLoadingSkills}
            complexity={complexity}
            onComplexityChange={setComplexity}
            onToggleSkill={toggleSkill}
          />
        )}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-border">
        <button
          onClick={step === 'roots' ? onClose : handleBack}
          className="flex items-center gap-1 px-3 py-1.5 text-sm rounded hover:bg-muted"
        >
          <ChevronLeft className="w-4 h-4" />
          {step === 'roots' ? 'Cancel' : 'Back'}
        </button>
        {step !== 'review' && (
          <button
            onClick={handleNext}
            disabled={selectedTopicIds.size === 0}
            className={cn(
              'flex items-center gap-1 px-3 py-1.5 text-sm rounded',
              selectedTopicIds.size > 0
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'bg-muted text-muted-foreground cursor-not-allowed',
            )}
          >
            Next
            <ChevronRight className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Copy Footer (always visible) */}
      <div className="px-4 py-3 border-t border-border bg-muted/30">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-muted-foreground">
            {copyableCount} skill{copyableCount !== 1 ? 's' : ''}
          </span>
          <div className="flex items-center gap-1">
            {FORMAT_OPTIONS.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => setFormat(option.value)}
                className={cn(
                  'flex items-center gap-1 px-2 py-1 text-[10px] rounded transition-colors',
                  format === option.value
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground',
                )}
                title={option.label}
              >
                {option.icon}
                <span className="hidden sm:inline">{option.label}</span>
              </button>
            ))}
          </div>
        </div>
        <button
          type="button"
          onClick={handleCopy}
          disabled={copyableCount === 0 || isCopying}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors',
            copySuccess
              ? 'bg-green-600 text-white'
              : 'bg-primary text-primary-foreground hover:bg-primary/90',
            (copyableCount === 0 || isCopying) && 'opacity-50 cursor-not-allowed',
          )}
        >
          {copySuccess ? (
            <>
              <Check className="h-4 w-4" />
              Copied!
            </>
          ) : isCopying ? (
            <>
              <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
              Copying...
            </>
          ) : (
            <>
              <Copy className="h-4 w-4" />
              Copy {copyableCount} skill{copyableCount !== 1 ? 's' : ''}
            </>
          )}
        </button>
      </div>
    </div>
  )
}

// --- Sub-components ---

function RootsStep({
  topics,
  selectedIds,
  childrenOf,
  onToggle,
  onToggleAll,
}: {
  topics: Topic[]
  selectedIds: Set<string>
  childrenOf: (id: string) => Topic[]
  onToggle: (id: string) => void
  onToggleAll: (ids: string[], selected: boolean) => void
}) {
  const allIds = useMemo(() => topics.map((t) => t.id), [topics])
  const allSelected = allIds.length > 0 && allIds.every((id) => selectedIds.has(id))

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium">Which areas are relevant?</h3>
        <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
          <input
            type="checkbox"
            checked={allSelected}
            onChange={() => onToggleAll(allIds, !allSelected)}
            className="rounded"
          />
          Select all
        </label>
      </div>
      {topics.length === 0 && (
        <p className="text-sm text-muted-foreground py-4 text-center">No topics created yet.</p>
      )}
      {topics.map((topic) => (
        <TopicCheckboxRow
          key={topic.id}
          topic={topic}
          checked={selectedIds.has(topic.id)}
          childCount={childrenOf(topic.id).length}
          onToggle={() => onToggle(topic.id)}
        />
      ))}
    </div>
  )
}

function DrilldownStep({
  children,
  selectedIds,
  breadcrumbs,
  childrenOf,
  onToggle,
  onToggleAll,
  onDrillInto,
}: {
  children: Topic[]
  selectedIds: Set<string>
  breadcrumbs: Array<{ id: string; name: string }>
  childrenOf: (id: string) => Topic[]
  onToggle: (id: string) => void
  onToggleAll: (ids: string[], selected: boolean) => void
  onDrillInto: (id: string) => void
}) {
  const childIds = useMemo(() => children.map((t) => t.id), [children])
  const allSelected = childIds.length > 0 && childIds.every((id) => selectedIds.has(id))

  return (
    <div className="space-y-1">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-1 mb-3 text-xs text-muted-foreground overflow-x-auto">
        {breadcrumbs.map((crumb, i) => (
          <span key={crumb.id} className="flex items-center gap-1 whitespace-nowrap">
            {i > 0 && <ChevronRight className="w-3 h-3 shrink-0" />}
            <span className={i === breadcrumbs.length - 1 ? 'text-foreground font-medium' : ''}>
              {crumb.name}
            </span>
          </span>
        ))}
      </div>

      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-medium">Narrow down</h3>
        <label className="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer">
          <input
            type="checkbox"
            checked={allSelected}
            onChange={() => onToggleAll(childIds, !allSelected)}
            className="rounded"
          />
          Select all
        </label>
      </div>
      {children.map((topic) => {
        const subChildren = childrenOf(topic.id)
        return (
          <div key={topic.id} className="flex items-center gap-1">
            <TopicCheckboxRow
              topic={topic}
              checked={selectedIds.has(topic.id)}
              childCount={subChildren.length}
              onToggle={() => onToggle(topic.id)}
              className="flex-1"
            />
            {subChildren.length > 0 && (
              <button
                onClick={() => onDrillInto(topic.id)}
                className="p-1 rounded hover:bg-muted text-muted-foreground"
                title="Drill into subtopics"
              >
                <ChevronRight className="w-4 h-4" />
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}

function ReviewStep({
  skillIds,
  deselectedIds,
  isLoading,
  complexity,
  onComplexityChange,
  onToggleSkill,
}: {
  skillIds: string[]
  deselectedIds: Set<string>
  isLoading: boolean
  complexity: Complexity
  onComplexityChange: (c: Complexity) => void
  onToggleSkill: (id: string) => void
}) {
  const activeCount = skillIds.filter((id) => !deselectedIds.has(id)).length

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="h-5 w-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        <span className="ml-2 text-sm text-muted-foreground">Loading skills...</span>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium mb-1">
          {activeCount} of {skillIds.length} skills selected
        </h3>
        <p className="text-xs text-muted-foreground">Toggle off any skills you don&apos;t need.</p>
      </div>

      {/* Complexity selector */}
      <div>
        <label className="text-xs font-medium text-muted-foreground mb-1 block">Complexity</label>
        <div className="flex items-center gap-1">
          {COMPLEXITY_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => onComplexityChange(opt.value)}
              className={cn(
                'px-3 py-1.5 text-xs rounded transition-colors',
                complexity === opt.value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-muted text-muted-foreground hover:bg-muted/80',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Skills list */}
      <div className="space-y-1">
        {skillIds.length === 0 && (
          <p className="text-sm text-muted-foreground py-4 text-center">
            No skills found for selected topics.
          </p>
        )}
        {skillIds.map((skillId) => (
          <label
            key={skillId}
            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted cursor-pointer"
          >
            <input
              type="checkbox"
              checked={!deselectedIds.has(skillId)}
              onChange={() => onToggleSkill(skillId)}
              className="rounded"
            />
            <span className="text-sm truncate">{skillId}</span>
          </label>
        ))}
      </div>
    </div>
  )
}

function TopicCheckboxRow({
  topic,
  checked,
  childCount,
  onToggle,
  className,
}: {
  topic: Topic
  checked: boolean
  childCount: number
  onToggle: () => void
  className?: string
}) {
  return (
    <label
      className={cn(
        'flex items-center gap-2 px-2 py-2 rounded hover:bg-muted cursor-pointer',
        className,
      )}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={onToggle}
        className="rounded shrink-0"
      />
      {topic.icon && <span className="text-sm shrink-0">{topic.icon}</span>}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium truncate">{topic.name}</div>
        {topic.description && (
          <div className="text-xs text-muted-foreground truncate">{topic.description}</div>
        )}
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        {topic.skills.length > 0 && (
          <span className="text-[10px] bg-muted px-1.5 py-0.5 rounded-full text-muted-foreground">
            {topic.skills.length} skill{topic.skills.length !== 1 ? 's' : ''}
          </span>
        )}
        {childCount > 0 && (
          <span className="text-[10px] bg-muted px-1.5 py-0.5 rounded-full text-muted-foreground">
            {childCount} sub
          </span>
        )}
      </div>
    </label>
  )
}

// Helper: get all descendant topic IDs
function getAllDescendants(topicId: string, topics: Topic[]): string[] {
  const result: string[] = []
  const queue = [topicId]
  while (queue.length > 0) {
    const current = queue.shift()
    if (current === undefined) continue
    const children = topics.filter((t) => t.parentTopicId === current)
    for (const child of children) {
      result.push(child.id)
      queue.push(child.id)
    }
  }
  return result
}

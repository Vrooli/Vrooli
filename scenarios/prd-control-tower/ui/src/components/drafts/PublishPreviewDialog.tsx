import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, CheckCircle, Loader2, Upload } from 'lucide-react'

import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Badge } from '../ui/badge'
import { DiffViewer } from './DiffViewer'
import { buildApiUrl } from '../../utils/apiClient'
import { useScenarioTemplates } from '../../hooks/useScenarioTemplates'
import type { Draft, PublishResponse, ScenarioExistenceResponse, ScenarioTemplate } from '../../types'

interface PublishPreviewDialogProps {
  draft: Draft
  open: boolean
  onClose: () => void
  onPublishSuccess: (result: PublishResponse) => void
  orphanedP0Count?: number
  orphanedP1Count?: number
}

type StepState = 'idle' | 'running' | 'done' | 'error'

type ProgressState = {
  scaffold: StepState
  write: StepState
  cleanup: StepState
}

const INITIAL_PROGRESS: ProgressState = {
  scaffold: 'idle',
  write: 'idle',
  cleanup: 'idle',
}

function humanizeScenarioName(name: string) {
  return name
    .split('-')
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
}

const progressSteps = [
  { key: 'scaffold' as const, label: 'Scaffold scenario from template' },
  { key: 'write' as const, label: 'Write PRD.md into repo' },
  { key: 'cleanup' as const, label: 'Clean up draft workspace' },
]

/**
 * PublishPreviewDialog orchestrates the publish UX.
 * For new PRDs it guides through template selection and scaffolding.
 * For existing PRDs it behaves like a diff + publish confirmation.
 */
export function PublishPreviewDialog({
  draft,
  open,
  onClose,
  onPublishSuccess,
  orphanedP0Count = 0,
  orphanedP1Count = 0,
}: PublishPreviewDialogProps) {
  const [publishedContent, setPublishedContent] = useState<string | null>(null)
  const [loadingPublished, setLoadingPublished] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [publishing, setPublishing] = useState(false)
  const [publishError, setPublishError] = useState<string | null>(null)
  const [progressState, setProgressState] = useState<ProgressState>(INITIAL_PROGRESS)
  const [templateValues, setTemplateValues] = useState<Record<string, string>>({})
  const [selectedTemplate, setSelectedTemplate] = useState<string>('')
  const [scenarioCheckLoading, setScenarioCheckLoading] = useState(false)
  const [scenarioCheckError, setScenarioCheckError] = useState<string | null>(null)
  const [existingScenarioInfo, setExistingScenarioInfo] = useState<ScenarioExistenceResponse | null>(null)
  const [confirmScenarioInput, setConfirmScenarioInput] = useState('')

  const isNewPRD = publishedContent === ''
  const { templates, loading: templatesLoading, error: templatesError } = useScenarioTemplates({ enabled: open && isNewPRD })
  const activeTemplate = useMemo<ScenarioTemplate | undefined>(
    () => templates.find((tpl) => tpl.name === selectedTemplate),
    [templates, selectedTemplate],
  )

  const hasUnlinkedTargets = orphanedP0Count > 0 || orphanedP1Count > 0
  const hasUnlinkedP0 = orphanedP0Count > 0

  // Reset component state when reopened
  useEffect(() => {
    if (!open) {
      return
    }
    setTemplateValues({
      SCENARIO_ID: draft.entity_name,
      SCENARIO_DISPLAY_NAME: humanizeScenarioName(draft.entity_name),
      SCENARIO_DESCRIPTION: '',
    })
    setSelectedTemplate('')
    setPublishError(null)
    setProgressState(INITIAL_PROGRESS)
    setExistingScenarioInfo(null)
    setScenarioCheckError(null)
    setScenarioCheckLoading(false)
    setConfirmScenarioInput('')
  }, [open, draft.entity_name])

  useEffect(() => {
    if (!selectedTemplate && templates.length > 0) {
      setSelectedTemplate(templates[0].name)
    }
  }, [templates, selectedTemplate])

  // Fetch published PRD content when dialog opens
  useEffect(() => {
    if (!open) {
      return
    }

    const fetchPublishedPRD = async () => {
      setLoadingPublished(true)
      setLoadError(null)
      setPublishedContent(null)

      try {
        const response = await fetch(buildApiUrl(`/catalog/${draft.entity_type}/${draft.entity_name}`))

        if (response.status === 404) {
          setPublishedContent('')
          setLoadingPublished(false)
          return
        }

        if (!response.ok) {
          throw new Error(`Failed to load published PRD: ${response.statusText}`)
        }

        const data = await response.json()
        setPublishedContent(data.content || '')
      } catch (err) {
        setLoadError(err instanceof Error ? err.message : 'Unknown error loading published PRD')
      } finally {
        setLoadingPublished(false)
      }
    }

    fetchPublishedPRD()
  }, [open, draft.entity_type, draft.entity_name])

  const updateTemplateValue = (key: string, value: string) => {
    setTemplateValues((prev) => ({ ...prev, [key]: value }))
  }

  useEffect(() => {
    if (!open || !isNewPRD) {
      return
    }

    const scenarioId = (templateValues.SCENARIO_ID || '').trim()
    setConfirmScenarioInput('')

    if (!scenarioId) {
      setExistingScenarioInfo(null)
      setScenarioCheckError(null)
      setScenarioCheckLoading(false)
      return
    }

    const controller = new AbortController()
    let active = true
    setScenarioCheckLoading(true)
    setScenarioCheckError(null)

    const fetchExistence = async () => {
      try {
        const response = await fetch(buildApiUrl(`/scenarios/${encodeURIComponent(scenarioId)}/exists`), {
          signal: controller.signal,
        })
        if (!response.ok) {
          throw new Error(`Failed to check scenario (${response.status})`)
        }
        const data: ScenarioExistenceResponse = await response.json()
        if (!active) {
          return
        }
        setExistingScenarioInfo(data.exists ? data : null)
      } catch (err) {
        if (!active || controller.signal.aborted) {
          return
        }
        setExistingScenarioInfo(null)
        setScenarioCheckError(err instanceof Error ? err.message : 'Unable to verify scenario ID')
      } finally {
        if (active) {
          setScenarioCheckLoading(false)
        }
      }
    }

    fetchExistence()

    return () => {
      active = false
      controller.abort()
    }
  }, [open, isNewPRD, templateValues.SCENARIO_ID])

  const requiredFieldsComplete = useMemo(() => {
    if (!activeTemplate) {
      return false
    }
    return activeTemplate.required_vars.every((field) => (templateValues[field.name] || '').trim().length > 0)
  }, [activeTemplate, templateValues])

  const scenarioIdValue = (templateValues.SCENARIO_ID || '').trim()
  const scenarioConflict = isNewPRD && Boolean(existingScenarioInfo?.exists)
  const overwriteConfirmed = !scenarioConflict || confirmScenarioInput.trim() === scenarioIdValue

  const canPublish =
    !publishing &&
    !loadingPublished &&
    !loadError &&
    (!isNewPRD || (activeTemplate && requiredFieldsComplete)) &&
    (!scenarioConflict || overwriteConfirmed) &&
    !scenarioCheckLoading

  const handlePublish = async () => {
    setPublishing(true)
    setPublishError(null)
    setProgressState({
      scaffold: isNewPRD ? 'running' : 'done',
      write: 'idle',
      cleanup: 'idle',
    })

    try {
      const payload: Record<string, unknown> = {
        create_backup: true,
      }
      const shouldForceTemplate = scenarioConflict && overwriteConfirmed

      if (isNewPRD && activeTemplate) {
        payload.delete_draft = true
        payload.template = {
          name: activeTemplate.name,
          variables: templateValues,
          force: shouldForceTemplate,
        }
      }

      const response = await fetch(buildApiUrl(`/drafts/${draft.id}/publish`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => null)
        throw new Error(errorData?.error || `Publish failed: ${response.statusText}`)
      }

      const result: PublishResponse = await response.json()
      setProgressState({
        scaffold: isNewPRD ? 'done' : 'done',
        write: 'done',
        cleanup: result.draft_removed ? 'done' : 'idle',
      })

      onPublishSuccess(result)
      onClose()
    } catch (err) {
      setPublishError(err instanceof Error ? err.message : 'Unexpected publish error')
      setProgressState((prev) => ({
        scaffold: prev.scaffold === 'running' ? 'error' : prev.scaffold,
        write: prev.write === 'running' ? 'error' : prev.write,
        cleanup: prev.cleanup,
      }))
    } finally {
      setPublishing(false)
    }
  }

  const hasChanges = publishedContent !== null && publishedContent !== draft.content

  const renderScenarioIdStatus = () => {
    if (!isNewPRD) {
      return null
    }
    if (!scenarioIdValue) {
      return <p className="text-xs text-muted-foreground">Enter a scenario ID to check availability.</p>
    }
    if (scenarioCheckLoading) {
      return (
        <p className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" /> Checking existing scenarios...
        </p>
      )
    }
    if (scenarioCheckError) {
      return <p className="text-xs text-amber-700">{scenarioCheckError}</p>
    }
    if (scenarioConflict && existingScenarioInfo?.path) {
      return (
        <p className="text-xs font-medium text-red-700">
          Scenario folder already exists at <code>{existingScenarioInfo.path}</code>
        </p>
      )
    }
    return <p className="text-xs text-emerald-700">Scenario ID available.</p>
  }

  const renderTemplateConfigurator = () => {
    if (!isNewPRD) {
      return (
        <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
          Publishing will update the existing scenario at{' '}
          <code className="rounded bg-white px-1 py-0.5 font-mono text-xs">
            {draft.entity_type}s/{draft.entity_name}/PRD.md
          </code>
          .
        </div>
      )
    }

    return (
      <div className="space-y-4 rounded-2xl border border-dashed border-slate-200 p-4">
        <div>
          <Label className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">Template</Label>
          {templatesLoading ? (
            <p className="text-sm text-muted-foreground">Loading templates...</p>
          ) : templatesError ? (
            <p className="text-sm text-red-600">{templatesError}</p>
          ) : (
            <div className="mt-2 flex flex-wrap gap-2">
              {templates.map((tpl) => (
                <Button
                  key={tpl.name}
                  size="sm"
                  variant={tpl.name === selectedTemplate ? 'default' : 'outline'}
                  onClick={() => setSelectedTemplate(tpl.name)}
                >
                  {tpl.display_name}
                </Button>
              ))}
              {templates.length === 0 && <p className="text-sm text-muted-foreground">No templates available.</p>}
            </div>
          )}
        </div>

        {activeTemplate && (
          <div className="space-y-4">
            <div>
              <p className="text-sm font-semibold text-slate-900">{activeTemplate.display_name}</p>
              <p className="text-xs text-muted-foreground">{activeTemplate.description}</p>
              {activeTemplate.stack?.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-2">
                  {activeTemplate.stack.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
            <div className="space-y-3">
              {activeTemplate.required_vars.map((field) => (
                <div key={field.name} className="space-y-1">
                  <Label className="text-xs font-medium text-slate-600">
                    {field.name}
                    <span className="ml-1 text-red-500">*</span>
                  </Label>
                  <Input
                    value={templateValues[field.name] || ''}
                    onChange={(event) => updateTemplateValue(field.name, event.target.value)}
                    placeholder={field.default || ''}
                  />
                  {field.name === 'SCENARIO_ID' && renderScenarioIdStatus()}
                  {field.description && <p className="text-xs text-muted-foreground">{field.description}</p>}
                </div>
              ))}
              {activeTemplate.optional_vars.length > 0 && (
                <details className="rounded-xl border border-slate-200 bg-white p-3 text-sm text-slate-600">
                  <summary className="cursor-pointer font-semibold text-slate-800">Optional parameters</summary>
                  <div className="mt-3 space-y-3">
                    {activeTemplate.optional_vars.map((field) => (
                      <div key={field.name} className="space-y-1">
                        <Label className="text-xs font-medium text-slate-600">{field.name}</Label>
                        <Input
                          value={templateValues[field.name] || ''}
                          onChange={(event) => updateTemplateValue(field.name, event.target.value)}
                          placeholder={field.default || ''}
                        />
                        {field.name === 'SCENARIO_ID' && renderScenarioIdStatus()}
                        {field.description && <p className="text-xs text-muted-foreground">{field.description}</p>}
                      </div>
                    ))}
                  </div>
                </details>
              )}
            </div>
            {scenarioConflict && (
              <div className="space-y-3 rounded-2xl border border-red-200 bg-red-50 p-4">
                <div className="flex items-start gap-2 text-sm text-red-800">
                  <AlertTriangle className="mt-0.5 h-4 w-4 text-red-600" />
                  <div>
                    <p className="font-semibold">Existing scenario will be overwritten</p>
                    <p>
                      Generating <code>{scenarioIdValue}</code> will replace files in{' '}
                      <code>{existingScenarioInfo?.path}</code>. Existing data may be lost.
                    </p>
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs font-semibold text-red-800">
                    Type <code>{scenarioIdValue}</code> to confirm overwrite
                  </Label>
                  <Input
                    value={confirmScenarioInput}
                    onChange={(event) => setConfirmScenarioInput(event.target.value)}
                    placeholder={scenarioIdValue}
                  />
                  {confirmScenarioInput && !overwriteConfirmed && (
                    <p className="text-xs text-red-700">Value must match exactly to proceed.</p>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  const renderDiffPane = () => {
    if (loadingPublished) {
      return (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" />
            <p className="mt-2 text-sm text-muted-foreground">Loading published PRD...</p>
          </div>
        </div>
      )
    }

    if (loadError) {
      return (
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <AlertTriangle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-sm text-destructive">{loadError}</p>
          </div>
        </div>
      )
    }

    if (isNewPRD) {
      return (
        <div className="space-y-2 text-sm text-slate-600">
          <p>No published PRD detected. The draft below becomes the canonical PRD.</p>
          <div className="max-h-96 overflow-y-auto rounded border bg-slate-50 p-3 text-xs font-mono">
            <pre className="whitespace-pre-wrap">{draft.content}</pre>
          </div>
        </div>
      )
    }

    if (!hasChanges) {
      return (
        <div className="flex h-64 flex-col items-center justify-center text-sm text-muted-foreground">
          <CheckCircle className="mb-2 h-8 w-8 text-green-600" />
          No changes detected between draft and published PRD.
        </div>
      )
    }

    return (
      <DiffViewer
        original={publishedContent || ''}
        modified={draft.content}
        title="Published PRD vs Draft Changes"
        height={460}
      />
    )
  }

  if (!open) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="flex max-h-[90vh] w-full max-w-6xl flex-col rounded-lg bg-white shadow-2xl">
        <div className="flex items-center justify-between border-b px-6 py-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-900">Publish Draft</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {isNewPRD ? 'Scaffold a new scenario and copy this draft into PRD.md.' : 'Review and publish changes to the canonical PRD.'}
            </p>
          </div>
          <Button variant="outline" onClick={onClose} disabled={publishing}>
            Cancel
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {hasUnlinkedTargets && (
            <div className={`rounded-lg border p-4 ${hasUnlinkedP0 ? 'border-red-200 bg-red-50' : 'border-amber-200 bg-amber-50'}`}>
              <div className="flex items-start gap-3">
                <AlertTriangle className={`h-5 w-5 mt-0.5 ${hasUnlinkedP0 ? 'text-red-600' : 'text-amber-600'}`} />
                <div className="flex-1 text-sm">
                  <p className={`font-medium ${hasUnlinkedP0 ? 'text-red-900' : 'text-amber-900'}`}>
                    {hasUnlinkedP0 ? 'Critical targets are not linked to requirements.' : 'Attention: unlinked P1 targets.'}
                  </p>
                  <p className={hasUnlinkedP0 ? 'text-red-700' : 'text-amber-700'}>
                    Link every P0 and P1 target to requirements before dispatching automation.
                  </p>
                </div>
              </div>
            </div>
          )}

          <div className="grid gap-6 lg:grid-cols-2">
            <div className="space-y-4">
              {renderTemplateConfigurator()}

              <div className="rounded-2xl border border-slate-200 p-4">
                <p className="text-sm font-semibold text-slate-800">Publish checklist</p>
                <div className="mt-3 space-y-2">
                  {progressSteps.map(({ key, label }) => {
                    const state = progressState[key]
                    return (
                      <div key={key} className="flex items-center gap-2 text-sm">
                        {state === 'done' && <CheckCircle className="h-4 w-4 text-green-600" />}
                        {state === 'running' && <Loader2 className="h-4 w-4 animate-spin text-primary" />}
                        {state === 'idle' && <span className="h-4 w-4 rounded-full border border-dashed border-slate-300" />}
                        {state === 'error' && <AlertTriangle className="h-4 w-4 text-red-600" />}
                        <span className="text-slate-700">{label}</span>
                      </div>
                    )
                  })}
                </div>
              </div>

              {publishError && (
                <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                  {publishError}
                </div>
              )}
            </div>

            <div className="rounded-2xl border border-slate-200 p-4 bg-white">
              {renderDiffPane()}
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between border-t bg-slate-50 px-6 py-4">
          <div className="text-xs text-muted-foreground">
            {isNewPRD ? 'Scenario files will be generated from the selected template.' : 'A backup of PRD.md will be stored before overwriting.'}
          </div>
          <div className="flex gap-3">
            <Button variant="outline" onClick={onClose} disabled={publishing}>
              Cancel
            </Button>
            <Button
              onClick={handlePublish}
              disabled={!canPublish || publishing}
              className="gap-2"
            >
              {publishing ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Publishing...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" />
                  {isNewPRD ? 'Create scenario & publish' : 'Publish to PRD.md'}
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

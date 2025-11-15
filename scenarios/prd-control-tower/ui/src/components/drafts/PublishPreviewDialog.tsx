import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle, Loader2, Upload } from 'lucide-react'
import { Button } from '../ui/button'
import { DiffViewer } from './DiffViewer'
import { buildApiUrl } from '../../utils/apiClient'
import type { Draft } from '../../types'

interface PublishPreviewDialogProps {
  draft: Draft
  open: boolean
  onClose: () => void
  onPublishSuccess: () => void
  orphanedP0Count?: number
  orphanedP1Count?: number
}

/**
 * PublishPreviewDialog shows a diff view of changes before publishing a draft to PRD.md
 * Includes validation, backup creation, and confirmation workflow.
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
          // No published PRD exists yet - this is a new PRD
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

  const handlePublish = async () => {
    setPublishing(true)
    setPublishError(null)

    try {
      const response = await fetch(buildApiUrl(`/drafts/${draft.id}/publish`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          create_backup: true, // Always create backup for safety
        }),
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => null)
        throw new Error(errorData?.error || `Publish failed: ${response.statusText}`)
      }

      // Success!
      onPublishSuccess()
      onClose()
    } catch (err) {
      setPublishError(err instanceof Error ? err.message : 'Unexpected publish error')
    } finally {
      setPublishing(false)
    }
  }

  if (!open) {
    return null
  }

  const isNewPRD = publishedContent === ''
  const hasChanges = publishedContent !== null && publishedContent !== draft.content
  const hasUnlinkedTargets = orphanedP0Count > 0 || orphanedP1Count > 0
  const hasUnlinkedP0 = orphanedP0Count > 0

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="flex max-h-[90vh] w-full max-w-6xl flex-col rounded-lg bg-white shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b px-6 py-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-900">Publish Preview</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              {isNewPRD
                ? 'Creating new PRD.md - no existing content to compare'
                : 'Review changes before publishing to PRD.md'}
            </p>
          </div>
          <Button variant="outline" onClick={onClose} disabled={publishing}>
            Cancel
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Unlinked Targets Warning */}
          {hasUnlinkedTargets && (
            <div className={`mb-4 rounded-lg border p-4 ${hasUnlinkedP0 ? 'border-red-200 bg-red-50' : 'border-amber-200 bg-amber-50'}`}>
              <div className="flex items-start gap-3">
                <AlertTriangle className={`h-5 w-5 mt-0.5 ${hasUnlinkedP0 ? 'text-red-600' : 'text-amber-600'}`} />
                <div className="flex-1">
                  <p className={`font-medium ${hasUnlinkedP0 ? 'text-red-900' : 'text-amber-900'}`}>
                    {hasUnlinkedP0 ? '⚠️ Critical Issue: Unlinked P0 Targets' : '⚠️ Warning: Unlinked P1 Targets'}
                  </p>
                  <p className={`mt-1 text-sm ${hasUnlinkedP0 ? 'text-red-700' : 'text-amber-700'}`}>
                    {orphanedP0Count > 0 && (
                      <span className="block">
                        <strong>{orphanedP0Count}</strong> P0 (critical) operational target{orphanedP0Count > 1 ? 's' : ''} lack{orphanedP0Count === 1 ? 's' : ''} requirement linkage
                      </span>
                    )}
                    {orphanedP1Count > 0 && (
                      <span className="block">
                        <strong>{orphanedP1Count}</strong> P1 operational target{orphanedP1Count > 1 ? 's' : ''} lack{orphanedP1Count === 1 ? 's' : ''} requirement linkage
                      </span>
                    )}
                  </p>
                  <p className={`mt-2 text-sm ${hasUnlinkedP0 ? 'text-red-700' : 'text-amber-700'}`}>
                    {hasUnlinkedP0
                      ? 'Publishing with unlinked P0 targets is strongly discouraged. Consider adding requirements or linking existing ones before publishing.'
                      : 'Consider linking these targets to requirements in the requirements/ folder before publishing.'
                    }
                  </p>
                </div>
              </div>
            </div>
          )}

          {loadingPublished ? (
            <div className="flex h-64 items-center justify-center">
              <div className="text-center">
                <Loader2 className="mx-auto h-8 w-8 animate-spin text-primary" />
                <p className="mt-2 text-sm text-muted-foreground">Loading published PRD...</p>
              </div>
            </div>
          ) : loadError ? (
            <div className="flex h-64 items-center justify-center">
              <div className="text-center">
                <AlertTriangle className="mx-auto h-8 w-8 text-destructive" />
                <p className="mt-2 text-sm text-destructive">{loadError}</p>
              </div>
            </div>
          ) : isNewPRD ? (
            <div className="space-y-4">
              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4">
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-blue-600 mt-0.5" />
                  <div>
                    <p className="font-medium text-blue-900">New PRD Creation</p>
                    <p className="mt-1 text-sm text-blue-700">
                      No existing PRD.md found. This will create a new PRD file at{' '}
                      <code className="rounded bg-blue-100 px-1 py-0.5 font-mono text-xs">
                        {draft.entity_type}s/{draft.entity_name}/PRD.md
                      </code>
                    </p>
                  </div>
                </div>
              </div>
              <div className="rounded-md border bg-slate-50 p-4">
                <p className="text-sm font-medium text-slate-700 mb-2">Draft Content Preview:</p>
                <div className="max-h-96 overflow-y-auto rounded bg-white p-3 text-xs font-mono">
                  <pre className="whitespace-pre-wrap text-slate-600">{draft.content}</pre>
                </div>
              </div>
            </div>
          ) : !hasChanges ? (
            <div className="flex h-64 items-center justify-center">
              <div className="text-center">
                <CheckCircle className="mx-auto h-8 w-8 text-green-600" />
                <p className="mt-2 text-sm text-muted-foreground">
                  No changes detected between draft and published PRD
                </p>
              </div>
            </div>
          ) : (
            <DiffViewer
              original={publishedContent || ''}
              modified={draft.content}
              title="Published PRD vs Draft Changes"
              height={500}
            />
          )}

          {publishError && (
            <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4">
              <div className="flex items-start gap-3">
                <AlertTriangle className="h-5 w-5 text-red-600 mt-0.5" />
                <div>
                  <p className="font-medium text-red-900">Publish Failed</p>
                  <p className="mt-1 text-sm text-red-700">{publishError}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t bg-slate-50 px-6 py-4">
          <div className="text-sm text-muted-foreground">
            {isNewPRD ? (
              <span>✨ New PRD will be created</span>
            ) : (
              <span>
                🔒 Backup will be created: <code className="rounded bg-slate-200 px-1 py-0.5 font-mono text-xs">PRD.md.backup.YYYYMMDD-HHMMSS</code>
              </span>
            )}
          </div>
          <div className="flex gap-3">
            <Button variant="outline" onClick={onClose} disabled={publishing}>
              Cancel
            </Button>
            <Button
              onClick={handlePublish}
              disabled={publishing || loadingPublished || !!loadError}
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
                  Publish to PRD.md
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

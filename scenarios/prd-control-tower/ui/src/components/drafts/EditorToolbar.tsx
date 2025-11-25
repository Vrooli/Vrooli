import { Loader2, Save, RotateCcw, RefreshCcw, Sparkles, Upload } from 'lucide-react'
import { Button } from '../ui/button'
import { Tooltip } from '../ui/tooltip'

interface EditorToolbarProps {
  saving: boolean
  refreshing: boolean
  hasUnsavedChanges: boolean
  onSave: () => void
  onDiscard: () => void
  onRefresh: () => void
  onOpenAI: () => void
  onPublish?: () => void
}

/**
 * Toolbar component for draft editor actions.
 * Handles save, discard, refresh, AI assist, and publish buttons.
 */
export function EditorToolbar({ saving, refreshing, hasUnsavedChanges, onSave, onDiscard, onRefresh, onOpenAI, onPublish }: EditorToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      {/* Primary Actions Group - High Visibility */}
      <div className="flex gap-2">
        <Tooltip content={!hasUnsavedChanges ? 'No changes to save' : 'Save your changes (⌘S / Ctrl+S)'}>
          <Button
            onClick={onSave}
            disabled={saving || !hasUnsavedChanges}
            size="lg"
            className="h-11 gap-2 font-semibold shadow-sm transition-all hover:shadow-md"
          >
            {saving ? <Loader2 size={18} className="animate-spin" /> : <Save size={18} />}
            <span>Save</span>
          </Button>
        </Tooltip>

        {onPublish && (
          <Tooltip content="Publish this draft to PRD.md in the repository">
            <Button
              variant="default"
              onClick={onPublish}
              size="lg"
              className="h-11 gap-2 bg-gradient-to-r from-green-600 to-green-700 font-semibold text-white shadow-sm transition-all hover:from-green-700 hover:to-green-800 hover:shadow-md"
            >
              <Upload size={18} />
              <span>Publish</span>
            </Button>
          </Tooltip>
        )}
      </div>

      {/* Divider for visual grouping */}
      <div className="h-8 w-px bg-slate-200" />

      {/* Secondary Actions Group */}
      <div className="flex gap-2">
        <Tooltip content="Get AI help with generating or improving sections (⌘K / Ctrl+K)">
          <Button variant="outline" onClick={onOpenAI} className="gap-2 border-violet-200 hover:bg-violet-50 hover:border-violet-300 transition-colors">
            <Sparkles size={16} className="text-violet-600" />
            <span className="hidden sm:inline">AI Assist</span>
            <span className="sm:hidden">AI</span>
          </Button>
        </Tooltip>

        <Tooltip content={!hasUnsavedChanges ? 'No changes to discard' : 'Discard unsaved changes'}>
          <Button variant="outline" onClick={onDiscard} disabled={!hasUnsavedChanges} className="gap-2">
            <RotateCcw size={16} />
            <span className="hidden sm:inline">Discard</span>
            <span className="sm:hidden">Undo</span>
          </Button>
        </Tooltip>

        <Tooltip content="Reload draft from server">
          <Button variant="ghost" onClick={onRefresh} disabled={refreshing} className="gap-2">
            {refreshing ? <Loader2 size={16} className="animate-spin" /> : <RefreshCcw size={16} />}
            <span className="hidden md:inline">Refresh</span>
          </Button>
        </Tooltip>
      </div>
    </div>
  )
}

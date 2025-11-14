import { Loader2, Save, RotateCcw, RefreshCcw, Sparkles, Upload } from 'lucide-react'
import { Button } from '../ui/button'

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
    <div className="flex flex-wrap gap-2">
      <Button onClick={onSave} disabled={saving || !hasUnsavedChanges}>
        {saving ? <Loader2 size={16} className="mr-2 animate-spin" /> : <Save size={16} className="mr-2" />}
        Save
      </Button>
      <Button variant="secondary" onClick={onDiscard} disabled={!hasUnsavedChanges}>
        <RotateCcw size={16} className="mr-2" /> Discard
      </Button>
      <Button variant="outline" onClick={onRefresh} disabled={refreshing}>
        {refreshing ? <Loader2 size={16} className="mr-2 animate-spin" /> : <RefreshCcw size={16} className="mr-2" />}
        Refresh
      </Button>
      <Button variant="outline" onClick={onOpenAI}>
        <Sparkles size={16} className="mr-2" /> AI Assist
      </Button>
      {onPublish && (
        <Button variant="default" onClick={onPublish} className="bg-green-600 hover:bg-green-700">
          <Upload size={16} className="mr-2" /> Publish
        </Button>
      )}
    </div>
  )
}

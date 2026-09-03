import { selectors } from '@/constants/selectors'

export interface EditorToolbarProps {
  editing: boolean
  onEditingChange: (editing: boolean) => void
  canUndo: boolean
  canRedo: boolean
  onUndo: () => void
  onRedo: () => void
  onReset: () => void
  /** The room currently selected in edit mode, if any. */
  selectedRoomLabel: string | null
  onRemoveSelected: () => void
  overrideCount: number
  saving: boolean
}

/** Edit mode chrome: enter/leave, undo/redo/reset, remove the selected room. */
export function EditorToolbar(props: EditorToolbarProps) {
  return (
    <div className="pointer-events-auto flex items-center gap-1 rounded-lg border border-border bg-background/85 px-2 py-1 text-xs shadow-sm backdrop-blur" data-testid={selectors.world.editor.toolbar} role="toolbar" aria-label="Layout editor">
      <button
        type="button"
        aria-pressed={props.editing}
        onClick={() => props.onEditingChange(!props.editing)}
        className={props.editing ? 'rounded-md bg-primary px-2 py-1 font-medium text-primary-foreground' : 'rounded-md border border-border px-2 py-1 font-medium hover:bg-muted'}
        data-testid={selectors.world.editor.toggle}
      >
        {props.editing ? 'Done' : 'Edit layout'}
      </button>
      {props.editing && (
        <>
          <button type="button" disabled={!props.canUndo} onClick={props.onUndo} className="rounded-md border border-border px-2 py-1 hover:bg-muted disabled:opacity-40" data-testid={selectors.world.editor.undo}>
            Undo
          </button>
          <button type="button" disabled={!props.canRedo} onClick={props.onRedo} className="rounded-md border border-border px-2 py-1 hover:bg-muted disabled:opacity-40" data-testid={selectors.world.editor.redo}>
            Redo
          </button>
          <button type="button" disabled={props.overrideCount === 0} onClick={props.onReset} className="rounded-md border border-border px-2 py-1 hover:bg-muted disabled:opacity-40" data-testid={selectors.world.editor.reset}>
            Reset layout
          </button>
          <button type="button" disabled={!props.selectedRoomLabel} onClick={props.onRemoveSelected} className="rounded-md border border-border px-2 py-1 hover:bg-muted disabled:opacity-40" data-testid={selectors.world.editor.remove}>
            Remove {props.selectedRoomLabel ?? 'room'}
          </button>
          <span className="ml-1 text-muted-foreground" data-testid={selectors.world.editor.status}>
            {props.overrideCount} change{props.overrideCount === 1 ? '' : 's'}
            {props.saving ? ' · saving…' : ''}
          </span>
          <span className="text-muted-foreground">Drag a room to move it</span>
        </>
      )}
    </div>
  )
}

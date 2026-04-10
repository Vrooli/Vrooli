import { StickyNote, Loader2 } from "lucide-react";
import { useState, useCallback, useEffect } from "react";

interface NoteEditorProps {
  note: string;
  onSave: (note: string) => Promise<void>;
  saving?: boolean;
}

export function NoteEditor({ note, onSave, saving }: NoteEditorProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(note);
  const [savedDraft, setSavedDraft] = useState<string | null>(null);

  // Sync draft when the prop updates (e.g., after refetch).
  useEffect(() => {
    if (!editing) {
      setDraft(note);
      setSavedDraft(null);
    }
  }, [note, editing]);

  const handleEdit = useCallback(() => {
    setDraft(savedDraft ?? note);
    setEditing(true);
  }, [note, savedDraft]);

  const handleSave = useCallback(async () => {
    await onSave(draft);
    setSavedDraft(draft);
    setEditing(false);
  }, [draft, onSave]);

  const handleCancel = useCallback(() => {
    setDraft(savedDraft ?? note);
    setEditing(false);
  }, [note, savedDraft]);

  // Show the optimistic value until the prop catches up.
  const displayNote = savedDraft ?? note;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium text-slate-400">
        <StickyNote className="h-3.5 w-3.5 text-amber-400/70" />
        Personal Note
      </div>

      {editing ? (
        <div className="space-y-2">
          <textarea
            className="w-full rounded-lg border border-white/10 bg-slate-900/50 px-3 py-2 text-sm text-slate-200 placeholder:text-slate-500 focus:border-cyan-500/50 focus:outline-none focus:ring-1 focus:ring-cyan-500/30"
            rows={3}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Add a note..."
            autoFocus
          />
          <div className="flex gap-2">
            <button
              className="rounded-md bg-cyan-600 px-3 py-1 text-xs font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : "Save"}
            </button>
            <button
              className="rounded-md px-3 py-1 text-xs font-medium text-slate-400 hover:text-slate-200"
              onClick={handleCancel}
              disabled={saving}
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button
          className="w-full rounded-lg border border-white/5 bg-slate-900/30 px-3 py-2 text-left text-sm hover:border-white/10 hover:bg-slate-900/50"
          onClick={handleEdit}
        >
          {displayNote ? (
            <span className="whitespace-pre-wrap text-slate-300">{displayNote}</span>
          ) : (
            <span className="text-slate-500">Add a note...</span>
          )}
        </button>
      )}
    </div>
  );
}

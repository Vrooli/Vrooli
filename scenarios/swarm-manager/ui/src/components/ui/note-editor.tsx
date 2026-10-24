import { StickyNote, Loader2 } from "lucide-react";
import { useState, useCallback, useEffect, useRef, useLayoutEffect } from "react";

import { errorMessageOf } from "../../lib/error-utils";

interface NoteEditorProps {
  note: string;
  onSave: (note: string) => Promise<void>;
  saving?: boolean;
}

export function NoteEditor({ note, onSave, saving }: NoteEditorProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(note);
  const [savedDraft, setSavedDraft] = useState<string | null>(null);

  const pendingRef = useRef(false);
  const ownerRevision = useRef(0);
  useLayoutEffect(() => {
    ownerRevision.current += 1;
  }, [note]);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const busy = pending || Boolean(saving);

  // An authoritative prop change supersedes the local accepted display.
  // Closing the editor alone must not restore a stale owner prop.
  useEffect(() => {
    setSavedDraft(null);
  }, [note]);

  useEffect(() => {
    if (!editing) setDraft(savedDraft ?? note);
  }, [note, editing, savedDraft]);

  const handleEdit = useCallback(() => {
    if (pendingRef.current || saving) return;
    setError(null);
    setDraft(savedDraft ?? note);
    setEditing(true);
  }, [note, savedDraft, saving]);

  const handleSave = useCallback(async () => {
    if (pendingRef.current || saving) return;
    pendingRef.current = true;
    setPending(true);
    setError(null);
    const submittedOwnerRevision = ownerRevision.current;
    try {
      await onSave(draft);
      // A caller may publish the canonical owner response before resolving.
      // Preserve that newer prop instead of restoring the submitted draft.
      setSavedDraft(ownerRevision.current === submittedOwnerRevision ? draft : null);
      setEditing(false);
    } catch (cause) {
      setError(errorMessageOf(cause, "Unable to save this note."));
    } finally {
      pendingRef.current = false;
      setPending(false);
    }
  }, [draft, onSave, saving]);

  const handleCancel = useCallback(() => {
    if (pendingRef.current || saving) return;
    setError(null);
    setDraft(savedDraft ?? note);
    setEditing(false);
  }, [note, savedDraft, saving]);

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
            disabled={busy}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Add a note..."
            autoFocus
          />
          {error && <p role="alert" className="text-sm text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button
              className="rounded-md bg-cyan-600 px-3 py-1 text-xs font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
              onClick={handleSave}
              aria-label={busy ? "Saving note" : "Save"}
              disabled={busy}
            >
              {busy ? <Loader2 className="h-3 w-3 animate-spin" /> : "Save"}
            </button>
            <button
              className="rounded-md px-3 py-1 text-xs font-medium text-slate-400 hover:text-slate-200"
              onClick={handleCancel}
              disabled={busy}
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button
          className="w-full rounded-lg border border-white/5 bg-slate-900/30 px-3 py-2 text-left text-sm hover:border-white/10 hover:bg-slate-900/50"
          onClick={handleEdit}
          disabled={busy}
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

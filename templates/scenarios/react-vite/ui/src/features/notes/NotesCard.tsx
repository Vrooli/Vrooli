import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { createNote, listNotes } from "../../lib/notes";

const NOTES_QUERY_KEY = ["notes"] as const;

/**
 * NotesCard is the canonical CRUD-reference feature: it lists notes,
 * creates new ones via mutation, and is the example new scenarios
 * copy when adding their own domain (tasks, users, …) to
 * `features/<name>/`.
 *
 * To replace this feature: delete `features/notes/`, drop the import
 * and render line in `App.tsx`, and follow the steps in
 * `docs/internal/REPLACING-NOTES.md`.
 */
export function NotesCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const notesQuery = useQuery({
    queryKey: NOTES_QUERY_KEY,
    queryFn: listNotes,
  });

  const createNoteMutation = useMutation({
    mutationFn: createNote,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: NOTES_QUERY_KEY });
    },
  });

  const handleCreateNote = () => {
    createNoteMutation.mutate({
      title: `Note ${(notesQuery.data?.notes.length ?? 0) + 1}`,
      body: "",
    });
  };

  return (
    <section
      data-testid={selectors.notes.card}
      aria-label={t(strings.notes.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.notes.title)}</h2>
      {notesQuery.isLoading && (
        <p data-testid={selectors.notes.loading} className="mt-2 text-slate-200">
          {t(strings.notes.loading)}
        </p>
      )}
      {notesQuery.error && (
        <p data-testid={selectors.notes.error} className="mt-2 text-red-400">
          {t(strings.notes.error)}
        </p>
      )}
      {notesQuery.data && notesQuery.data.notes.length === 0 && (
        <p data-testid={selectors.notes.empty} className="mt-2 text-slate-200">
          {t(strings.notes.empty)}
        </p>
      )}
      {notesQuery.data && notesQuery.data.notes.length > 0 && (
        <ul data-testid={selectors.notes.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {notesQuery.data.notes.map((note) => (
            <li key={note.id}>{note.title}</li>
          ))}
        </ul>
      )}
      <Button
        data-testid={selectors.notes.createButton}
        className="mt-4"
        onClick={handleCreateNote}
        disabled={createNoteMutation.isPending}
      >
        {t(strings.notes.create)}
      </Button>
    </section>
  );
}

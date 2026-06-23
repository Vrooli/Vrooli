import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { MessageInitShape } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { CreateNoteRequestSchema } from "@vrooli/proto-types/network-manager/v1/notes/notes_pb";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { notesClient } from "../../api/notes";
import { errorMessage } from "../../lib/errorMessage";
import { AttachmentUpload } from "./AttachmentUpload";

const NOTES_QUERY_KEY = ["notes"] as const;

/**
 * NotesCard is the canonical CRUD-reference feature: it lists notes,
 * creates new ones via mutation, and is the example new scenarios
 * copy when adding their own domain (tasks, users, …) to
 * `features/<name>/`.
 *
 * To replace this feature: copy this folder for your real domain, get
 * the new feature green, then delete `features/notes/` and drop the
 * import/render line in `App.tsx`.
 */
export function NotesCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const notesQuery = useQuery({
    queryKey: NOTES_QUERY_KEY,
    queryFn: () => notesClient.listNotes({}),
  });

  const createNoteMutation = useMutation({
    mutationFn: (input: MessageInitShape<typeof CreateNoteRequestSchema>) => notesClient.createNote(input),
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
          {errorMessage(notesQuery.error, t)}
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
            <li key={note.id} className="rounded-lg border border-white/10 p-3">
              <div className="font-medium">{note.title}</div>
              {note.createdAt && (
                <div data-testid={selectors.notes.createdAt} className="mt-1 text-xs text-slate-400">
                  {t(strings.notes.createdAtLabel)}{" "}
                  {formatDate(timestampDate(note.createdAt), {
                    dateStyle: "medium",
                    timeStyle: "short",
                  })}
                </div>
              )}
              <div data-testid={selectors.notes.attachmentCount} className="mt-1 text-xs text-slate-400">
                {t(strings.notes.attachmentsLabel, { count: note.attachmentKeys.length })}
              </div>
              <AttachmentUpload noteId={note.id} />
            </li>
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
      {createNoteMutation.error && (
        <p data-testid={selectors.notes.error} className="mt-2 text-red-400">
          {errorMessage(createNoteMutation.error, t)}
        </p>
      )}
    </section>
  );
}

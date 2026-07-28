import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { MessageInitShape } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { CreateNoteRequestSchema } from "@vrooli/proto-types/channel-manager/v1/notes/notes_pb";
import type { Note } from "../../api/notes";

import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { DataTable, type DataTableColumn } from "../../components/ui/data-table";
import { EmptyState } from "../../components/ui/empty-state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { notesClient } from "../../api/notes";
import { errorMessage } from "../../lib/errorMessage";
import { AttachmentUpload } from "./AttachmentUpload";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";

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

  const columns: Array<DataTableColumn<Note>> = [
    {
      id: "title",
      header: t(strings.notes.table.title),
      accessor: (note) => <span className="font-medium">{note.title}</span>,
      sortValue: (note) => note.title,
      searchValue: (note) => note.title,
    },
    {
      id: "created",
      header: t(strings.notes.table.created),
      accessor: (note) => note.createdAt && (
        <span data-testid={selectors.notes.createdAt}>
          {formatDate(timestampDate(note.createdAt), {
            dateStyle: "medium",
            timeStyle: "short",
          })}
        </span>
      ),
      sortValue: (note) => note.createdAt ? timestampDate(note.createdAt).getTime() : 0,
      searchValue: (note) => note.createdAt ? timestampDate(note.createdAt).toISOString() : "",
    },
    {
      id: "attachments",
      header: t(strings.notes.table.attachments),
      accessor: (note) => (
        <span data-testid={selectors.notes.attachmentCount}>
          {t(strings.notes.attachmentsLabel, { count: note.attachmentKeys.length })}
        </span>
      ),
      sortValue: (note) => note.attachmentKeys.length,
      searchValue: (note) => String(note.attachmentKeys.length),
    },
    {
      id: "upload",
      header: t(strings.notes.table.actions),
      accessor: (note) => <AttachmentUpload noteId={note.id} />,
    },
  ];
  const experienceState: ExperienceSurfaceState = notesQuery.isLoading
    ? "loading"
    : notesQuery.error
    ? "error"
    : notesQuery.data?.notes.length === 0
    ? "empty"
    : "ready";
  const experienceStatus = notesQuery.isLoading
    ? t(strings.notes.loading)
    : notesQuery.error
    ? errorMessage(notesQuery.error, t)
    : undefined;

  return (
    <ExperienceSurface
      surfaceId="notes"
      state={experienceState}
      statusMessage={experienceStatus}
      data-testid={selectors.notes.surface}
    >
      <Card data-testid={selectors.notes.card} aria-label={t(strings.notes.title)}>
        <CardHeader className="flex-row items-center justify-between gap-3">
          <CardTitle>{t(strings.notes.title)}</CardTitle>
          <Button
            data-testid={selectors.notes.createButton}
            size="sm"
            onClick={handleCreateNote}
            disabled={createNoteMutation.isPending}
          >
            {t(strings.notes.create)}
          </Button>
        </CardHeader>
        <CardContent>
        {notesQuery.isLoading && (
          <p data-testid={selectors.notes.loading} className="text-sm text-app-muted-foreground">
            {t(strings.notes.loading)}
          </p>
        )}
        {notesQuery.error && (
          <p data-testid={selectors.notes.error} className="text-sm text-app-danger">
            {errorMessage(notesQuery.error, t)}
          </p>
        )}
        {notesQuery.data && notesQuery.data.notes.length === 0 && (
          <div data-testid={selectors.notes.empty}>
            <EmptyState title={t(strings.notes.empty)} />
          </div>
        )}
        {notesQuery.data && notesQuery.data.notes.length > 0 && (
          <DataTable
            rows={notesQuery.data.notes}
            columns={columns}
            getRowKey={(note) => note.id}
            caption={t(strings.notes.title)}
            searchPlaceholder={t(strings.notes.table.search)}
            emptyMessage={t(strings.notes.empty)}
            tableTestId={selectors.notes.list}
          />
        )}
        {createNoteMutation.error && (
          <p data-testid={selectors.notes.error} className="mt-3 text-sm text-app-danger">
            {errorMessage(createNoteMutation.error, t)}
          </p>
        )}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}

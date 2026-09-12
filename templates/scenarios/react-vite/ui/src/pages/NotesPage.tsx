import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { NotesCard } from "../features/notes/NotesCard";
import { NotesMeasureCard } from "../features/notes/NotesMeasureCard";
import { useTranslation } from "../i18n";

export function NotesPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.notes} aria-labelledby="notes-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="notes-heading" title={t(strings.pages.notes.title)} description={t(strings.pages.notes.description)} />
      <div className="grid gap-space-sm xl:grid-cols-[minmax(0,18rem)_minmax(0,1fr)]">
        <NotesMeasureCard />
        <NotesCard />
      </div>
    </section>
  );
}

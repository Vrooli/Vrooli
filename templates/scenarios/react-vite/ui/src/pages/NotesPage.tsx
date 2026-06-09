import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { NotesCard } from "../features/notes/NotesCard";
import { NotesMeasureCard } from "../features/notes/NotesMeasureCard";
import { useTranslation } from "../i18n";

export function NotesPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.notes}
      aria-labelledby="notes-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="notes-heading" className="text-2xl font-semibold">
        {t(strings.pages.notes.title)}
      </h2>
      <NotesMeasureCard />
      <NotesCard />
    </section>
  );
}

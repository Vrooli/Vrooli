import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { EditorCard } from "../features/editor/EditorCard";
import { OpStackCard } from "../features/editor/OpStackCard";
import { useTranslation } from "../i18n";

export function EditorPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.editor}
      aria-labelledby="editor-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="editor-heading" className="text-2xl font-semibold">
        {t(strings.pages.editor.title)}
      </h2>
      <EditorCard />
      <OpStackCard />
    </section>
  );
}

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { Workspace } from "../features/workspace/Workspace";
import { useTranslation } from "../i18n";

/**
 * Workspace route — the unified canvas + inspector + history surface where the
 * loaded image flows across Edit / Enhance / Create / Analyze modes.
 */
export function WorkspacePage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.workspace}
      aria-labelledby="workspace-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="workspace-heading" className="text-2xl font-semibold">
        {t(strings.pages.workspace.title)}
      </h2>
      <Workspace />
    </section>
  );
}

import { Wrench } from "lucide-react";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";

/**
 * Skills index — placeholder surface.
 *
 * TODO: replace with the real skills index/detail surfaces once the
 * skill-catalog Connect-RPC client lands. The plan defers Phases 5 & 6
 * for this exact reason — `feedback_no_git_mutations` + plan §11 says
 * "do not stub" the backend, so we keep the navigation surface area
 * reserved and render an honest empty state.
 */
export function SkillsPlaceholder() {
  const { t } = useTranslation();
  return (
    <section className="flex flex-col gap-4">
      <PanelHeader title={t(strings.nav.skillsTodo)} />
      <EmptyState
        icon={<Wrench className="h-8 w-8" aria-hidden />}
        title={t(strings.nav.skillsTodo)}
        description={t(strings.settings.catalogSyncPending)}
      />
    </section>
  );
}

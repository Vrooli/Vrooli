import { FileCog } from "lucide-react";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";

/**
 * Manifests index — placeholder surface.
 *
 * TODO: replace with the real manifests index + editor once the
 * manifests Connect-RPC client lands.
 */
export function ManifestsPlaceholder() {
  const { t } = useTranslation();
  return (
    <section className="flex flex-col gap-4">
      <PanelHeader title={t(strings.nav.manifestsTodo)} />
      <EmptyState
        icon={<FileCog className="h-8 w-8" aria-hidden />}
        title={t(strings.nav.manifestsTodo)}
        description={t(strings.settings.catalogSyncPending)}
      />
    </section>
  );
}

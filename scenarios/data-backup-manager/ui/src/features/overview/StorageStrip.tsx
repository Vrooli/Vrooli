import { HardDrive } from "lucide-react";

import { UsageBar } from "../../components/ui/usage-bar";
import { StatusChip } from "../../components/ui/status-chip";
import { AsyncSection } from "../../components/AsyncSection";
import { EmptyState } from "../../components/EmptyState";
import { useDestinations } from "../../hooks/useDestinations";
import { backendSlug, capPolicySlug, usageMeta } from "../../lib/status";
import { BACKEND_STRINGS, CAP_POLICY_STRINGS, USAGE_STRINGS } from "../../consts/statusStrings";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * One usage bar per destination, colored by the server's usage_state. Shows the
 * cap policy and the (read-only) encryption algorithm so an operator can see at
 * a glance that storage is encrypted and how close each repo is to its cap.
 */
export function StorageStrip() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = useDestinations();
  const destinations = data ?? [];

  return (
    <section data-testid={selectors.overview.storage} className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-app-foreground">{t(strings.overview.storageHeading)}</h2>
      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={destinations.length === 0}
        onRetry={() => void refetch()}
        emptyState={
          <EmptyState
            icon={HardDrive}
            title={t(strings.overview.storageEmpty)}
            data-testid={selectors.overview.storageEmpty}
          />
        }
      >
        <ul className="grid gap-3 sm:grid-cols-2">
          {destinations.map((d) => {
            const usage = usageMeta(d.usageState);
            return (
              <li
                key={d.id}
                className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-app-foreground">{d.name}</p>
                    <p className="truncate text-xs text-app-muted-foreground">
                      {t(BACKEND_STRINGS[backendSlug(d.backendKind)])}
                    </p>
                  </div>
                  <StatusChip tone={usage.tone} labelKey={USAGE_STRINGS[usage.slug]} />
                </div>
                <UsageBar usageBytes={d.usageBytes} capBytes={d.capBytes} usageState={d.usageState} />
                <p className="text-xs text-app-muted-foreground">
                  {t(CAP_POLICY_STRINGS[capPolicySlug(d.capPolicy)])}
                </p>
              </li>
            );
          })}
        </ul>
      </AsyncSection>
    </section>
  );
}

import { useState } from "react";
import { FolderTree, HardDrive, ShieldCheck } from "lucide-react";

import { Button } from "../../components/ui/button";
import { StatusChip } from "../../components/ui/status-chip";
import {
  useDestinationSuggestions,
  useDismissSuggestion,
  useInvalidateSuggestions,
  useTargetSuggestions,
} from "../../hooks/useSuggestions";
import { useRegisterTarget } from "../../hooks/useTargets";
import { useCreateDestination } from "../../hooks/useDestinations";
import { BackendKind, CapPolicy } from "../../api/destinations";
import type { DestinationSuggestion, TargetSuggestion } from "../../api/discovery";
import { driveClassMeta, sourceKindSlug } from "../../lib/status";
import { DRIVE_CLASS_STRINGS, SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { formatBytes } from "../../lib/format";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Onboarding Suggestions panel. Surfaces discovered targets worth protecting and
 * destinations worth backing up to, each with one-click **Enable** (which calls
 * the existing register/create RPCs — never a separate "accept" path) and
 * **Dismiss**. When `onboarding` is set it leads with the cold-start headline;
 * otherwise it's a quiet helper above the storage strip.
 */
export function SuggestionsPanel({ onboarding = false }: { onboarding?: boolean }) {
  const { t } = useTranslation();
  const targets = useTargetSuggestions();
  const destinations = useDestinationSuggestions();
  const dismiss = useDismissSuggestion();
  const register = useRegisterTarget();
  const createDestination = useCreateDestination();
  const invalidate = useInvalidateSuggestions();

  // Which suggestion is mid-enable, so only its button shows the busy state.
  const [activeId, setActiveId] = useState<string | null>(null);

  const targetList = targets.data ?? [];
  const destinationList = destinations.data ?? [];

  const enableTarget = async (s: TargetSuggestion) => {
    setActiveId(s.id);
    try {
      await register.mutateAsync({
        owner: s.owner,
        name: s.name,
        sourceKind: s.sourceKind,
        locator: s.locator,
      });
      invalidate();
    } finally {
      setActiveId(null);
    }
  };

  const enableDestination = async (s: DestinationSuggestion) => {
    setActiveId(s.id);
    try {
      await createDestination.mutateAsync({
        name: s.label,
        backendKind: BackendKind.FILESYSTEM,
        location: s.location,
        capBytes: 0n,
        // Server defaults UNSPECIFIED → ALERT_BLOCK (cap-on-by-default policy).
        capPolicy: CapPolicy.UNSPECIFIED,
      });
      invalidate();
    } finally {
      setActiveId(null);
    }
  };

  const enableLabel = (id: string) =>
    t(activeId === id ? strings.discovery.enabling : strings.discovery.enable);

  return (
    <section
      data-testid={selectors.discovery.panel}
      aria-labelledby="discovery-heading"
      className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div id="discovery-heading" className="flex items-start gap-3">
        <ShieldCheck aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-primary" />
        <div className="flex flex-col gap-1">
          <h2 className="text-sm font-semibold text-app-foreground">
            {t(onboarding ? strings.discovery.onboardingTitle : strings.discovery.title)}
          </h2>
          <p className="text-sm text-app-muted-foreground">
            {t(onboarding ? strings.discovery.onboardingBody : strings.discovery.subtitle)}
          </p>
        </div>
      </div>

      {/* Targets to protect */}
      <div data-testid={selectors.discovery.targetsGroup} className="flex flex-col gap-2">
        <h3 className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          <FolderTree aria-hidden="true" className="h-4 w-4" />
          {t(strings.discovery.targetsHeading)}
        </h3>
        {targetList.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.discovery.targetsEmpty)}</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {targetList.map((s) => {
              const kind = sourceKindSlug(s.sourceKind);
              return (
                <li
                  key={s.id}
                  data-testid={selectors.discovery.suggestionRow({ id: s.id })}
                  className="flex flex-wrap items-start justify-between gap-3 rounded-panel border border-app-border bg-app-surface-muted p-3"
                >
                  <div className="min-w-0 flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-app-foreground">
                        {s.owner}/{s.name}
                      </span>
                      <StatusChip tone="info" labelKey={SOURCE_KIND_STRINGS[kind]} />
                      {s.approxBytes > 0n && (
                        <span className="text-xs text-app-muted-foreground">
                          {t(strings.discovery.approxSize, { size: formatBytes(s.approxBytes) })}
                        </span>
                      )}
                    </div>
                    <p className="truncate font-mono text-xs text-app-muted-foreground">{s.locator}</p>
                    <p className="text-xs text-app-muted-foreground">{s.rationale}</p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button
                      size="sm"
                      data-testid={selectors.discovery.enableButton({ id: s.id })}
                      disabled={activeId === s.id}
                      onClick={() => void enableTarget(s)}
                    >
                      {enableLabel(s.id)}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      data-testid={selectors.discovery.dismissButton({ id: s.id })}
                      onClick={() => dismiss.mutate(s.id)}
                    >
                      {t(strings.discovery.dismiss)}
                    </Button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {/* Destinations to back up to */}
      <div data-testid={selectors.discovery.destinationsGroup} className="flex flex-col gap-2">
        <h3 className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          <HardDrive aria-hidden="true" className="h-4 w-4" />
          {t(strings.discovery.destinationsHeading)}
        </h3>
        {destinationList.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">
            {t(strings.discovery.destinationsEmpty)}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {destinationList.map((s) => {
              const cls = driveClassMeta(s.driveClass);
              return (
                <li
                  key={s.id}
                  data-testid={selectors.discovery.suggestionRow({ id: s.id })}
                  className="flex flex-wrap items-start justify-between gap-3 rounded-panel border border-app-border bg-app-surface-muted p-3"
                >
                  <div className="min-w-0 flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-app-foreground">
                        {s.label}
                      </span>
                      <StatusChip tone={cls.tone} labelKey={DRIVE_CLASS_STRINGS[cls.slug]} />
                    </div>
                    <p className="truncate font-mono text-xs text-app-muted-foreground">
                      {s.location}
                    </p>
                    <p className="text-xs text-app-muted-foreground">
                      {t(strings.discovery.freeOfTotal, {
                        free: formatBytes(s.freeBytes),
                        total: formatBytes(s.totalBytes),
                      })}
                    </p>
                    <p className="text-xs text-app-muted-foreground">{s.rationale}</p>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1">
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        data-testid={selectors.discovery.enableButton({ id: s.id })}
                        disabled={activeId === s.id || !s.separateRootOk}
                        onClick={() => void enableDestination(s)}
                      >
                        {enableLabel(s.id)}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        data-testid={selectors.discovery.dismissButton({ id: s.id })}
                        onClick={() => dismiss.mutate(s.id)}
                      >
                        {t(strings.discovery.dismiss)}
                      </Button>
                    </div>
                    {!s.separateRootOk && (
                      <span className="text-xs text-app-warning">{t(strings.discovery.unsafe)}</span>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

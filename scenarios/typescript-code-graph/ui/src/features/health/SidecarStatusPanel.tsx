import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useSidecarHealth } from "./useSidecarHealth";
import { sidecarStatusMeta } from "./sidecarStatus";

/**
 * Persistent diagnostic panel for the Node ts-morph sidecar. Reads the
 * `sidecar_status` / `sidecar_message` fields off the existing /health
 * response (no dedicated RPC) and renders state + last message. Read-only.
 * Status is conveyed by icon + label (color is redundant), satisfying the
 * non-color-only A11y contract.
 */
export function SidecarStatusPanel() {
  const { t } = useTranslation();
  const { data, isPending, isError } = useSidecarHealth();

  const meta = sidecarStatusMeta(data?.sidecarStatus ?? 0);
  const Icon = meta.icon;
  const message = data?.sidecarMessage ?? "";

  return (
    <section
      data-testid={selectors.features.sidecar.root}
      data-status={meta.token}
      aria-label={t(strings.sidecar.title)}
      className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3 backdrop-blur-sm"
    >
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
          {t(strings.sidecar.title)}
        </span>
        {isPending ? (
          <span
            data-testid={selectors.features.sidecar.loading}
            className="text-xs text-app-muted-foreground"
          >
            {t(strings.sidecar.loading)}
          </span>
        ) : isError ? (
          <span
            data-testid={selectors.features.sidecar.error}
            className="text-xs text-app-danger"
          >
            {t(strings.sidecar.unreachable)}
          </span>
        ) : (
          <span
            data-testid={selectors.features.sidecar.indicator({ status: meta.token })}
            className={cn("flex items-center gap-1.5 text-sm font-medium", meta.accentClass)}
          >
            <Icon aria-hidden="true" className="h-4 w-4" />
            {t(meta.labelKey)}
          </span>
        )}
      </div>
      {message.length > 0 ? (
        <p
          data-testid={selectors.features.sidecar.message}
          className="text-xs text-app-muted-foreground"
        >
          {t(strings.sidecar.lastMessage)} <span className="text-app-foreground">{message}</span>
        </p>
      ) : null}
    </section>
  );
}

import { Check, X } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsentWeight, DeploymentTier, type SafetyPolicy } from "../../api/safety";
import { useSafetyPolicy } from "../safety/useSafetyPolicy";

const TIER_LABEL = {
  [DeploymentTier.UNSPECIFIED]: strings.settings.responsibleUse.tier.unspecified,
  [DeploymentTier.LOCAL]: strings.settings.responsibleUse.tier.local,
  [DeploymentTier.PUBLIC]: strings.settings.responsibleUse.tier.public,
} as const;

const WEIGHT_LABEL = {
  [ConsentWeight.UNSPECIFIED]: strings.settings.responsibleUse.weight.unspecified,
  [ConsentWeight.NONE]: strings.settings.responsibleUse.weight.none,
  [ConsentWeight.LOW]: strings.settings.responsibleUse.weight.low,
  [ConsentWeight.HIGH]: strings.settings.responsibleUse.weight.high,
} as const;

/** A single enforced-control row with an on/off pill. */
function ControlRow({ label, on }: { label: string; on: boolean }) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-between gap-3 py-1 text-sm">
      <span className="text-app-foreground">{label}</span>
      <span
        className={
          on
            ? "inline-flex items-center gap-1 rounded-pill bg-app-success/15 px-2 py-0.5 text-xs font-medium text-app-success"
            : "inline-flex items-center gap-1 rounded-pill bg-app-surface-muted px-2 py-0.5 text-xs font-medium text-app-muted-foreground"
        }
      >
        {on ? (
          <Check aria-hidden="true" className="h-3 w-3" />
        ) : (
          <X aria-hidden="true" className="h-3 w-3" />
        )}
        {on ? t(strings.settings.responsibleUse.on) : t(strings.settings.responsibleUse.off)}
      </span>
    </div>
  );
}

/** The resolved-policy body once the fetch succeeds. */
function PolicyBody({ policy }: { policy: SafetyPolicy }) {
  const { t } = useTranslation();
  const tierLabel = t(TIER_LABEL[policy.tier]);

  return (
    <div className="flex flex-col gap-4">
      <div
        data-testid={selectors.responsibleUse.tier}
        className="flex items-center justify-between gap-3"
      >
        <span className="text-sm text-app-muted-foreground">
          {t(strings.settings.responsibleUse.tierLabel)}
        </span>
        <span className="rounded-pill bg-app-primary/10 px-2 py-0.5 text-sm font-medium text-app-primary">
          {tierLabel}
        </span>
      </div>

      <div className="flex flex-col">
        <h4 className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.settings.responsibleUse.controlsHeading)}
        </h4>
        <ControlRow label={t(strings.settings.responsibleUse.consent)} on={policy.requireConsent} />
        <ControlRow label={t(strings.settings.responsibleUse.nsfwScan)} on={policy.forceNsfwScan} />
        <ControlRow
          label={t(strings.settings.responsibleUse.provenance)}
          on={policy.requireProvenance}
        />
        <div className="flex items-center justify-between gap-3 py-1 text-sm">
          <span className="text-app-foreground">
            {policy.rateLimitPerMin > 0
              ? t(strings.settings.responsibleUse.rateLimit, { count: policy.rateLimitPerMin })
              : t(strings.settings.responsibleUse.rateLimitNone)}
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <h4 className="text-xs font-semibold uppercase text-app-muted-foreground">
          {t(strings.settings.responsibleUse.weightsHeading)}
        </h4>
        {policy.opWeights.length > 0 ? (
          <table data-testid={selectors.responsibleUse.weights} className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-app-muted-foreground">
                <th className="py-1 font-medium">
                  {t(strings.settings.responsibleUse.weightOperation)}
                </th>
                <th className="py-1 text-right font-medium">
                  {t(strings.settings.responsibleUse.weightLevel)}
                </th>
              </tr>
            </thead>
            <tbody>
              {policy.opWeights.map((w) => (
                <tr key={w.operation} className="border-t border-app-border">
                  <td className="py-1 font-mono text-xs text-app-foreground">{w.operation}</td>
                  <td className="py-1 text-right text-app-muted-foreground">
                    {t(WEIGHT_LABEL[w.weight])}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-sm text-app-muted-foreground">
            {t(strings.settings.responsibleUse.empty)}
          </p>
        )}
      </div>

      {policy.summary && (
        <p
          data-testid={selectors.responsibleUse.summary}
          className="rounded-control border border-app-border bg-app-surface-muted p-3 text-xs text-app-muted-foreground"
        >
          {policy.summary}
        </p>
      )}
    </div>
  );
}

/**
 * Read-only Responsible-Use panel. Fetches `SafetyService.GetPolicy` and shows
 * the resolved deployment tier, the enforced controls (consent / NSFW scan /
 * provenance / rate limit), the per-op consent-weight table, and the policy
 * summary. Informational only — the tier is a deploy-time server setting, not a
 * UI toggle.
 */
export function ResponsibleUsePanel() {
  const { t } = useTranslation();
  const query = useSafetyPolicy();

  return (
    <section
      data-testid={selectors.responsibleUse.panel}
      aria-labelledby="responsible-use-heading"
      className="flex flex-col gap-4"
    >
      <h3
        id="responsible-use-heading"
        className="text-sm font-semibold uppercase text-app-muted-foreground"
      >
        {t(strings.pages.settings.responsibleUseHeading)}
      </h3>

      {query.isLoading ? (
        <p
          data-testid={selectors.responsibleUse.loading}
          className="text-sm text-app-muted-foreground"
        >
          {t(strings.settings.responsibleUse.loading)}
        </p>
      ) : query.error || !query.data ? (
        <p data-testid={selectors.responsibleUse.error} className="text-sm text-app-danger">
          {t(strings.settings.responsibleUse.error)}
        </p>
      ) : (
        <PolicyBody policy={query.data} />
      )}
    </section>
  );
}

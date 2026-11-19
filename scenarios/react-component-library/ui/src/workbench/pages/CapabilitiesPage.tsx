import { useState } from "react";
import { Check, CircleAlert, Copy, Play, RefreshCw, Sparkles } from "lucide-react";

import { useTranslation } from "../../i18n";
import { formatRelative, useCapabilities } from "../data";
import {
  Banner,
  Chip,
  Dot,
  Empty,
  Kpi,
  Panel,
  PanelBody,
  PanelHead,
  Skeleton,
  WbButton,
} from "../primitives";

export function CapabilitiesPage() {
  const { t } = useTranslation();
  const capabilities = useCapabilities();
  const [copied, setCopied] = useState<string | null>(null);

  const states = capabilities.data?.states ?? [];
  const available = states.filter((state) => state.status === "available").length;
  const degraded = states.filter((state) => state.status === "degraded").length;
  const unavailable = states.filter((state) => state.status === "unavailable").length;
  const checked = states[0]?.checkedAt ? new Date(states[0].checkedAt) : undefined;

  const copy = async (command: string) => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(command);
      window.setTimeout(() => setCopied(null), 1600);
    } catch {
      setCopied(null);
    }
  };

  return (
    <div className="wb-page" data-testid="capabilities-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.capabilities.kicker", { defaultValue: "Library · Capabilities" })}</p>
          <h1 className="wb-h1">{t("workbench.capabilities.title", { defaultValue: "What the library can rely on" })}</h1>
          <p className="wb-lede">
            {t("workbench.capabilities.lede", {
              defaultValue:
                "Integrations that unlock assisted extraction, shared experience contracts and provenance. A missing service is a visible finding, not a skipped check.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <Chip tone={unavailable > 0 ? "amber" : "green"}>
            {t("workbench.capabilities.checked", {
              defaultValue: "Checked {{age}}",
              age: formatRelative(checked),
            })}
          </Chip>
          <WbButton variant="ghost" size="md" onClick={() => void capabilities.refetch()} disabled={capabilities.isFetching}>
            <RefreshCw aria-hidden size={14} />
            {t("workbench.capabilities.refresh", { defaultValue: "Re-check" })}
          </WbButton>
        </div>
      </div>

      {capabilities.isError ? (
        <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
          <b>{t("workbench.capabilities.errorTitle", { defaultValue: "The capability registry did not answer." })}</b>{" "}
          {t("workbench.capabilities.errorBody", {
            defaultValue: "Recovery commands cannot be shown without the registry; nothing is assumed available.",
          })}
        </Banner>
      ) : null}

      <div className="wb-kpis">
        <Kpi label={t("workbench.capabilities.kpiAvailable", { defaultValue: "Available" })} value={available} tone="green" detail={t("workbench.capabilities.kpiAvailableDetail", { defaultValue: "Ready to use" })} />
        <Kpi label={t("workbench.capabilities.kpiDegraded", { defaultValue: "Degraded" })} value={degraded} tone="amber" detail={t("workbench.capabilities.kpiDegradedDetail", { defaultValue: "Partial capability" })} />
        <Kpi label={t("workbench.capabilities.kpiUnavailable", { defaultValue: "Unavailable" })} value={unavailable} tone="red" detail={t("workbench.capabilities.kpiUnavailableDetail", { defaultValue: "Recovery required" })} />
        <Kpi label={t("workbench.capabilities.kpiTotal", { defaultValue: "Declared" })} value={states.length} detail={t("workbench.capabilities.kpiTotalDetail", { defaultValue: "Optional integrations" })} />
      </div>

      {capabilities.isLoading ? (
        <div className="wb-grid-2">
          <Skeleton height={180} />
          <Skeleton height={180} />
          <Skeleton height={180} />
          <Skeleton height={180} />
        </div>
      ) : states.length === 0 ? (
        <Empty title={t("workbench.capabilities.emptyTitle", { defaultValue: "No capabilities declared" })}>
          {t("workbench.capabilities.emptyBody", {
            defaultValue: "The registry returned no integrations. This is an empty registry, not a healthy one.",
          })}
        </Empty>
      ) : (
        <div className="wb-grid-2">
          {states.map((state) => (
            <Panel key={state.id}>
              <PanelHead
                title={
                  <>
                    <Sparkles aria-hidden size={14} style={{ color: "var(--wb-signal)" }} />
                    {state.name}
                  </>
                }
                meta={state.dependencyKind}
              />
              <PanelBody>
                <div className="wb-row" style={{ flexWrap: "wrap", marginBottom: 10 }}>
                  <Chip
                    tone={state.status === "available" ? "green" : state.status === "degraded" ? "amber" : "red"}
                  >
                    <Dot tone={state.status === "available" ? "green" : state.status === "degraded" ? "amber" : "red"} />
                    {state.status}
                  </Chip>
                  {state.required ? <Chip tone="ice">{t("workbench.capabilities.required", { defaultValue: "required" })}</Chip> : null}
                  <span className="wb-small wb-muted-2">{state.message}</span>
                </div>
                <p className="wb-small wb-muted" style={{ lineHeight: 1.6 }}>
                  {state.description}
                </p>
                <div style={{ marginTop: 10 }}>
                  <span className="wb-caption">{t("workbench.capabilities.enables", { defaultValue: "What it enables" })}</span>
                  <p className="wb-small wb-muted" style={{ marginTop: 3 }}>
                    {state.features?.length ? state.features.join(" · ") : state.description}
                  </p>
                </div>
                {state.operatorCommand ? (
                  <div style={{ marginTop: 10 }}>
                    <span className="wb-caption">{t("workbench.capabilities.recovery", { defaultValue: "Recovery" })}</span>
                    <div className="wb-row" style={{ marginTop: 4 }}>
                      <code
                        className="wb-mono wb-small"
                        style={{
                          flex: 1,
                          minWidth: 0,
                          overflow: "auto",
                          padding: "7px 9px",
                          border: "1px solid var(--wb-line)",
                          borderRadius: 7,
                          background: "#0b0a0f",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {state.operatorCommand}
                      </code>
                      <button
                        type="button"
                        className="wb-iconbtn"
                        aria-label={t("workbench.capabilities.copyCommand", { defaultValue: "Copy recovery command" })}
                        onClick={() => void copy(state.operatorCommand ?? "")}
                      >
                        {copied === state.operatorCommand ? <Check aria-hidden size={14} /> : <Copy aria-hidden size={14} />}
                      </button>
                    </div>
                  </div>
                ) : null}
                {state.status !== "available" ? (
                  <div style={{ marginTop: 10 }}>
                    <WbButton variant="ghost" size="sm" disabled>
                      <Play aria-hidden size={12} />
                      {t("workbench.capabilities.start", { defaultValue: "Start service" })}
                    </WbButton>
                    <p className="wb-muted-2 wb-small" style={{ marginTop: 6 }}>
                      {t("workbench.capabilities.startNote", {
                        defaultValue:
                          "Starting a scenario is an operator action. Copy the command above; the workbench does not run it for you.",
                      })}
                    </p>
                  </div>
                ) : null}
              </PanelBody>
            </Panel>
          ))}
        </div>
      )}
    </div>
  );
}

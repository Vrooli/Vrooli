import { useQuery } from "@tanstack/react-query";
import { fetchScenarios } from "../../api/selection";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import { i18n } from "../../i18n";

export function StepOperatingMode({ selected, overrides, onAutoRestart, target = "local" }: { selected: Set<string>; overrides?: Record<string, { autoRestart?: boolean }>; onAutoRestart: (name: string, enabled: boolean) => void; target?: string }) {
  const { data, isError, isFetching, refetch } = useQuery({ queryKey: ["selection-scenarios", target], queryFn: () => fetchScenarios(target), retry: false });
  const scenarios = (data?.scenarios ?? []).filter((scenario) => scenario.systemRequired || selected.has(scenario.name));
  const alwaysOn = scenarios.filter((scenario) => scenario.autoRestart);
  const onDemand = scenarios.filter((scenario) => !scenario.autoRestart);
  const renderGroup = (label: string, group: typeof scenarios) => <SettingsList.Group label={label}>
    {group.map((scenario) => {
      const autoRestart = overrides?.[scenario.name]?.autoRestart ?? scenario.autoRestart;
      const overridden = overrides?.[scenario.name]?.autoRestart !== undefined;
      return <div key={scenario.name} data-testid="operating-mode-row"><SettingsList.Row label={<span data-testid="operating-mode-row-name">{scenario.name}</span>} hint={<span data-testid="recommendation-note" role="note">{overridden ? i18n.t("onboarding.mode.override", { recommendation: scenario.autoRestart ? i18n.t("onboarding.mode.always") : i18n.t("onboarding.mode.demand") }) : i18n.t("onboarding.mode.recommendation", { recommendation: scenario.autoRestart ? i18n.t("onboarding.mode.always") : i18n.t("onboarding.mode.demand") })}{overridden && <span className="block text-primary-soft" data-testid="override-indicator">{i18n.t("onboarding.mode.saved")}</span>}</span>}>
        <Switch checked={autoRestart} disabled={isError || isFetching} onChange={(event) => onAutoRestart(scenario.name, event.currentTarget.checked)} aria-label={i18n.t("onboarding.mode.keep", { name: scenario.name })} data-testid="keep-running-toggle" />
      </SettingsList.Row></div>;
    })}
  </SettingsList.Group>;
  return <div data-testid="step-operating-mode">
    <p className="surface-eyebrow">{i18n.t("onboarding.mode.eyebrow")}</p>
    <h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.mode.heading")}</h1>
    <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.mode.intro")}</p>
    {isError && <div className="mt-4 rounded-xl border border-warning/30 bg-warning-surface p-4" data-testid="scenario-list-unavailable" role="alert">
      <p className="text-sm text-foreground">{i18n.t("onboarding.mode.catalogError")}</p>
      <button type="button" className="mt-3 inline-flex min-h-11 items-center rounded px-3 text-sm text-warning underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus" onClick={() => void refetch()} disabled={isFetching}>
        {i18n.t(isFetching ? "onboarding.mode.retryingCatalog" : "onboarding.mode.retryCatalog")}
      </button>
    </div>}
    <SettingsList className="mt-6" variant="auto" density="comfortable">
      {scenarios.length === 0 && !isError && <SettingsList.Group label={i18n.t("onboarding.mode.loading")}><SettingsList.Row label={i18n.t("onboarding.mode.operatingMode")} hint={<span data-testid="recommendation-note" role="note">{i18n.t("onboarding.mode.loadingRecommendations")}</span>}><Switch disabled aria-label={i18n.t("onboarding.mode.keepSelected")} data-testid="keep-running-toggle" /></SettingsList.Row></SettingsList.Group>}
      {alwaysOn.length > 0 && renderGroup(i18n.t("onboarding.mode.alwaysOn"), alwaysOn)}
      {onDemand.length > 0 && renderGroup(i18n.t("onboarding.mode.onDemand"), onDemand)}
    </SettingsList>
  </div>;
}

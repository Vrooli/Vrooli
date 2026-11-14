import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useScenario } from "./scenarioContextValue";

/**
 * Scenario selector shared by every per-scenario workflow. Reads its options
 * from the fleet scan (via ScenarioContext) so the list is always real,
 * discoverable scenarios — never a hardcoded enum.
 */
export function ScenarioPicker() {
  const { t } = useTranslation();
  const { scenario, setScenario, scenarios, isLoadingScenarios } = useScenario();

  return (
    <label className="flex items-center gap-2 text-sm">
      <span className="font-medium text-app-muted-foreground">
        {t(strings.perf.picker.label)}
      </span>
      <select
        data-testid={selectors.perf.scenarioSelect}
        value={scenario}
        disabled={isLoadingScenarios && scenarios.length === 0}
        onChange={(e) => setScenario(e.target.value)}
        aria-label={t(strings.perf.picker.label)}
        className="min-w-[12rem] rounded-control border border-app-border bg-app-surface px-2 py-1 text-app-foreground"
      >
        {scenarios.map((s) => (
          <option key={s} value={s}>
            {s}
          </option>
        ))}
      </select>
    </label>
  );
}

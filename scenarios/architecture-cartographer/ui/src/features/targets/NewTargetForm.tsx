import * as React from "react";
import { Link } from "react-router-dom";

import { ApiError } from "../../api/client";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import type { PreferenceStorage } from "../../hooks/usePersistedPreference";
import { useExtractGraph } from "./controllers/useTargetsController";
import { useRecentTargets } from "./hooks/useRecentTargets";

const SCENARIO_INVALID = /[\s/\\]/;

interface ValidationResult {
  ok: boolean;
  errorKey?: typeof strings.pages.newTarget.validationRequired
    | typeof strings.pages.newTarget.validationInvalid;
}

function validateScenarioName(raw: string): ValidationResult {
  const trimmed = raw.trim();
  if (trimmed.length === 0) {
    return { ok: false, errorKey: strings.pages.newTarget.validationRequired };
  }
  if (SCENARIO_INVALID.test(trimmed)) {
    return { ok: false, errorKey: strings.pages.newTarget.validationInvalid };
  }
  return { ok: true };
}

export interface NewTargetFormProps {
  /** Inject when tests need to assert recent-target persistence side-effects. */
  recentTargetsStorage?: PreferenceStorage;
  /** Override timestamp generation in tests. */
  now?: () => Date;
}

/**
 * Form for starting a fresh `ExtractGraph` run against a target scenario.
 * Records the scenario in `useRecentTargets` on success and surfaces a
 * "Open workspace" affordance that takes the user straight into the target.
 */
export function NewTargetForm({ recentTargetsStorage, now }: NewTargetFormProps) {
  const { t } = useTranslation();

  const [scenario, setScenario] = React.useState("");
  const [submittedScenario, setSubmittedScenario] = React.useState<string | null>(null);
  const [validationError, setValidationError] = React.useState<string | null>(null);

  const extract = useExtractGraph();
  const recent = useRecentTargets({ storage: recentTargetsStorage, now });

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (extract.isPending) return;
    const result = validateScenarioName(scenario);
    if (!result.ok) {
      setValidationError(result.errorKey ? t(result.errorKey) : "");
      return;
    }
    setValidationError(null);
    const trimmed = scenario.trim();
    setSubmittedScenario(trimmed);
    extract.mutate(
      { scenario: trimmed },
      {
        onSuccess: () => {
          recent.record(trimmed);
        },
      },
    );
  };

  const snapshot = extract.data?.snapshot;
  const apiError = extract.error instanceof ApiError ? extract.error : null;
  const unknownError = extract.error && !apiError ? extract.error : null;
  const targetScenario = submittedScenario ?? scenario.trim();

  return (
    <form
      data-testid={selectors.features.targets.newForm.root}
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4"
      noValidate
    >
      <div className="flex flex-col gap-2">
        <label htmlFor="new-target-scenario" className="text-sm font-medium text-app-foreground">
          {t(strings.pages.newTarget.scenarioPathLabel)}
        </label>
        <input
          id="new-target-scenario"
          name="scenario"
          type="text"
          autoComplete="off"
          spellCheck={false}
          value={scenario}
          onChange={(event) => {
            setScenario(event.target.value);
            if (validationError !== null) setValidationError(null);
          }}
          data-testid={selectors.features.targets.newForm.scenarioInput}
          aria-invalid={validationError !== null}
          aria-describedby="new-target-scenario-hint"
          className="rounded-control border border-app-border bg-app-background px-3 py-2 text-sm text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
        />
        <p id="new-target-scenario-hint" className="text-xs text-app-muted-foreground">
          {t(strings.pages.newTarget.scenarioPathHint)}
        </p>
        {validationError !== null && (
          <p
            data-testid={selectors.features.targets.newForm.scenarioInputError}
            className="text-xs text-app-danger"
            role="alert"
          >
            {validationError}
          </p>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="submit"
          disabled={extract.isPending}
          data-testid={selectors.features.targets.newForm.submitButton}
        >
          {extract.isPending
            ? t(strings.pages.newTarget.submitting)
            : t(strings.pages.newTarget.submitButton)}
        </Button>
        {snapshot && targetScenario.length > 0 && (
          <Button
            asChild
            variant="outline"
            data-testid={selectors.features.targets.newForm.openWorkspaceLink}
          >
            <Link to={`/targets/${encodeScenarioPath(targetScenario)}`}>
              {t(strings.pages.newTarget.openWorkspace)}
            </Link>
          </Button>
        )}
      </div>

      {snapshot && (
        <p
          data-testid={selectors.features.targets.newForm.successBanner}
          role="status"
          className="rounded-control border border-app-primary/40 bg-app-primary/10 px-3 py-2 text-sm text-app-foreground"
        >
          {extract.data?.fromCache
            ? t(strings.pages.newTarget.fromCacheMessage, { snapshotId: snapshot.id })
            : t(strings.pages.newTarget.successMessage, {
                snapshotId: snapshot.id,
                durationMs: Number(snapshot.extractionMs),
              })}
        </p>
      )}

      {(apiError ?? unknownError) && (
        <p
          data-testid={selectors.features.targets.newForm.errorBanner}
          role="alert"
          className="rounded-control border border-app-danger/40 bg-app-danger/10 px-3 py-2 text-sm text-app-foreground"
        >
          {apiError ? apiError.message : t(strings.errors.internal)}
        </p>
      )}
    </form>
  );
}

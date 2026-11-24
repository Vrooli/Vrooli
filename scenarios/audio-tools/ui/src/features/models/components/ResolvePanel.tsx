import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "../../../components/ui/button";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Panel } from "../../../components/ui/panel";
import { Select } from "../../../components/ui/select";
import { StatusDot } from "../../../components/ui/status-dot";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import {
  isNotFound,
  primarySubtag,
  resolveModel,
  type ModelRow,
} from "../../../services/models";
import {
  CAPABILITIES,
  capabilityLabel,
  readinessLabelKey,
  readinessTone,
} from "../modelDisplay";
import { Detail } from "./Detail";

/**
 * Resolve panel — asks ModelRegistryService.ResolveModel for the model the
 * registry would actually pick. NOT_FOUND is a valid policy outcome ("nothing
 * eligible"), so it renders as a message rather than an error.
 */
export function ResolvePanel() {
  const { t } = useTranslation();
  const [capability, setCapability] = useState<string>(CAPABILITIES[0]);
  const [language, setLanguage] = useState("");
  const [engineId, setEngineId] = useState("");
  const [commercial, setCommercial] = useState(false);
  const resolve = useMutation({ mutationFn: resolveModel });

  function run() {
    resolve.mutate({
      capability,
      language: primarySubtag(language),
      engineId: capability === "stt" ? engineId.trim() : "",
      commercial,
    });
  }

  const result = resolve.data;

  return (
    <Panel
      title={t(strings.models.resolveTitle)}
      description={t(strings.models.resolveDescription)}
      aria-label={t(strings.models.resolveTitle)}
    >
      <form
        className="grid gap-3 md:grid-cols-2 xl:grid-cols-5 xl:items-end"
        onSubmit={(event) => {
          event.preventDefault();
          run();
        }}
      >
        <div className="flex flex-col gap-1">
          <label
            htmlFor="models-resolve-capability"
            className="text-xs font-medium text-app-muted-foreground"
          >
            {t(strings.models.resolveCapabilityLabel)}
          </label>
          <Select
            id="models-resolve-capability"
            value={capability}
            onChange={(event) => {
              const next = event.target.value;
              setCapability(next);
              if (next !== "stt") setEngineId("");
            }}
          >
            {CAPABILITIES.map((value) => (
              <option key={value} value={value}>
                {capabilityLabel(value, t)}
              </option>
            ))}
          </Select>
        </div>

        {capability === "stt" ? (
          <div className="flex flex-col gap-1">
            <label
              htmlFor="models-resolve-engine"
              className="text-xs font-medium text-app-muted-foreground"
            >
              {t(strings.models.resolveEngineLabel)}
            </label>
            <Input
              id="models-resolve-engine"
              value={engineId}
              placeholder={t(strings.models.resolveEnginePlaceholder)}
              onChange={(event) => setEngineId(event.target.value)}
            />
          </div>
        ) : null}

        <div className="flex flex-col gap-1">
          <label
            htmlFor="models-resolve-language"
            className="text-xs font-medium text-app-muted-foreground"
          >
            {t(strings.models.resolveLanguageLabel)}
          </label>
          <Input
            id="models-resolve-language"
            value={language}
            placeholder={t(strings.models.resolveLanguagePlaceholder)}
            onChange={(event) => setLanguage(event.target.value)}
          />
        </div>

        <div className="flex items-end pb-2">
          <label
            htmlFor="models-resolve-commercial"
            className="flex items-center gap-2 text-sm text-app-foreground"
          >
            <input
              id="models-resolve-commercial"
              type="checkbox"
              checked={commercial}
              onChange={(event) => setCommercial(event.target.checked)}
              className="h-4 w-4 rounded border-app-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/50"
            />
            {t(strings.models.resolveCommercialLabel)}
          </label>
        </div>

        <div className="flex items-end">
          <Button type="submit" disabled={resolve.isPending}>
            {resolve.isPending
              ? t(strings.common.loading)
              : t(strings.models.resolveButton)}
          </Button>
        </div>
      </form>

      <div className="mt-4" aria-live="polite">
        {resolve.isPending ? (
          <p className="text-sm text-app-muted-foreground">
            {t(strings.common.loading)}
          </p>
        ) : !result ? (
          <p className="text-sm text-app-muted-foreground">
            {t(strings.models.resolveEmpty)}
          </p>
        ) : !result.ok ? (
          isNotFound(result.error) ? (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.models.resolveNotFound)}
            </p>
          ) : (
            <ApiErrorState error={result.error} onRetry={run} />
          )
        ) : (
          <ResolvedModelView
            model={result.data.model}
            explanation={result.data.explanation}
          />
        )}
      </div>
    </Panel>
  );
}

function ResolvedModelView({
  model,
  explanation,
}: {
  model: ModelRow;
  explanation: string;
}) {
  const { t } = useTranslation();
  const dash = t(strings.common.dash);

  return (
    <div className="rounded-panel border border-app-border bg-app-surface-muted/40 p-3">
      <h3 className="text-sm font-semibold text-app-foreground">
        {t(strings.models.resolveResultTitle)}
      </h3>
      <div className="mt-2 flex flex-wrap items-baseline gap-2">
        <span className="text-sm font-medium">{model.name || model.id}</span>
        <span className="font-mono text-xs text-app-muted-foreground">
          {model.id}
        </span>
      </div>
      <dl className="mt-2 grid gap-2 text-xs sm:grid-cols-3">
        <Detail label={t(strings.models.colReadiness)}>
          <StatusDot
            tone={readinessTone(model.readiness)}
            label={t(readinessLabelKey(model.readiness))}
          />
        </Detail>
        <Detail label={t(strings.models.colLicense)}>
          {model.license || dash}
        </Detail>
        <Detail label={t(strings.models.colRuntime)}>
          {model.runtime || dash}
        </Detail>
      </dl>
      <div className="mt-3">
        <p className="text-xs font-medium text-app-muted-foreground">
          {t(strings.models.resolveExplanation)}
        </p>
        <p className="mt-1 text-sm text-app-foreground">{explanation}</p>
      </div>
    </div>
  );
}

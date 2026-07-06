import { useMutation } from "@tanstack/react-query";
import { Play, Route } from "lucide-react";
import { useState } from "react";

import {
  defaultPreviewInput,
  previewRoute,
  privacyOptions,
  profileOptions,
  type PreviewRouteInput,
} from "../api/gateway";
import { StatusChip } from "../components/StatusChip";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const numberValue = (value: string, fallback: number) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

export function RoutePreviewPage() {
  const { t } = useTranslation();
  const [form, setForm] = useState<PreviewRouteInput>(defaultPreviewInput);
  const previewMutation = useMutation({
    mutationFn: previewRoute,
  });

  const update = <K extends keyof PreviewRouteInput>(key: K, value: PreviewRouteInput[K]) => {
    setForm((current) => ({ ...current, [key]: value }));
  };

  const selected = previewMutation.data?.candidates.find((candidate) => candidate.selected);

  return (
    <section
      data-testid={selectors.pages.routePreview}
      aria-labelledby="route-preview-heading"
      className="grid gap-5 xl:grid-cols-[minmax(320px,420px)_1fr]"
    >
      <div className="flex flex-col gap-4">
        <header className="flex flex-col gap-2">
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.routePreview.eyebrow)}
          </p>
          <h2 id="route-preview-heading" className="text-2xl font-semibold">
            {t(strings.pages.routePreview.title)}
          </h2>
          <p className="text-sm text-app-muted-foreground">
            {t(strings.pages.routePreview.description)}
          </p>
        </header>

        <form
          data-testid={selectors.routePreview.form}
          aria-label={t(strings.pages.routePreview.formLabel)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
          onSubmit={(event) => {
            event.preventDefault();
            previewMutation.mutate(form);
          }}
        >
          <div className="grid gap-3">
            <label className="grid gap-1 text-sm font-medium">
              {t(strings.pages.routePreview.fields.scenario)}
              <input
                data-testid={selectors.routePreview.scenarioInput}
                value={form.scenario}
                onChange={(event) => update("scenario", event.target.value)}
                className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 font-mono text-sm"
              />
            </label>
            <label className="grid gap-1 text-sm font-medium">
              {t(strings.pages.routePreview.fields.operation)}
              <input
                value={form.operation}
                onChange={(event) => update("operation", event.target.value)}
                className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 font-mono text-sm"
              />
            </label>
            <label className="grid gap-1 text-sm font-medium">
              {t(strings.pages.routePreview.fields.role)}
              <input
                value={form.role}
                onChange={(event) => update("role", event.target.value)}
                className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 font-mono text-sm"
              />
            </label>
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="grid gap-1 text-sm font-medium">
                {t(strings.pages.routePreview.fields.profile)}
                <select
                  data-testid={selectors.routePreview.profileSelect}
                  value={form.profile}
                  onChange={(event) => update("profile", numberValue(event.target.value, form.profile))}
                  className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                >
                  {profileOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
              <label className="grid gap-1 text-sm font-medium">
                {t(strings.pages.routePreview.fields.privacy)}
                <select
                  value={form.privacyClass}
                  onChange={(event) => update("privacyClass", numberValue(event.target.value, form.privacyClass))}
                  className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                >
                  {privacyOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <div className="grid gap-3 sm:grid-cols-3">
              <label className="grid gap-1 text-sm font-medium">
                {t(strings.pages.routePreview.fields.timeout)}
                <input
                  type="number"
                  value={form.timeoutMs}
                  onChange={(event) => update("timeoutMs", numberValue(event.target.value, form.timeoutMs))}
                  className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                />
              </label>
              <label className="grid gap-1 text-sm font-medium">
                {t(strings.pages.routePreview.fields.maxCost)}
                <input
                  type="number"
                  step="0.01"
                  value={form.maxCostUsd}
                  onChange={(event) => update("maxCostUsd", numberValue(event.target.value, form.maxCostUsd))}
                  className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                />
              </label>
              <label className="grid gap-1 text-sm font-medium">
                {t(strings.pages.routePreview.fields.maxTokens)}
                <input
                  type="number"
                  value={form.maxOutputTokens}
                  onChange={(event) => update("maxOutputTokens", numberValue(event.target.value, form.maxOutputTokens))}
                  className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                />
              </label>
            </div>
          </div>
          <button
            type="submit"
            data-testid={selectors.routePreview.submit}
            className="mt-4 inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-control bg-app-primary px-4 text-sm font-semibold text-app-primary-foreground"
          >
            <Route aria-hidden="true" size={17} />
            {previewMutation.isPending ? t(strings.states.loading) : t(strings.pages.routePreview.preview)}
          </button>
        </form>
      </div>

      <div className="flex flex-col gap-4">
        {previewMutation.isError ? (
          <div data-testid={selectors.routePreview.error} className="rounded-panel border border-red-200 bg-red-50 p-4 text-sm text-red-700">
            {errorMessage(previewMutation.error, t)}
          </div>
        ) : null}

        <div
          data-testid={selectors.routePreview.result}
          role="region"
          aria-label={t(strings.pages.routePreview.selectedRoute)}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 className="font-semibold">{t(strings.pages.routePreview.selectedRoute)}</h3>
              <p className="mt-1 text-sm text-app-muted-foreground">
                {t(strings.pages.routePreview.selectedRouteDescription)}
              </p>
            </div>
            <button
              type="button"
              disabled={!previewMutation.data?.valid}
              className="inline-flex min-h-10 items-center gap-2 rounded-control border border-app-border px-3 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Play aria-hidden="true" size={16} />
              {t(strings.pages.routePreview.execute)}
            </button>
          </div>

          {previewMutation.data ? (
            <div className="mt-4 grid gap-3 md:grid-cols-3">
              <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
                <p className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.routePreview.selectedProvider)}</p>
                <p className="mt-1 font-mono text-sm">{previewMutation.data.selectedProvider || t(strings.states.none)}</p>
              </div>
              <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
                <p className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.routePreview.routePlan)}</p>
                <p className="mt-1 font-mono text-sm">{previewMutation.data.routePlanId || t(strings.states.none)}</p>
              </div>
              <div className="rounded-control border border-app-border bg-app-surface-muted p-3">
                <p className="text-xs uppercase text-app-muted-foreground">{t(strings.pages.routePreview.fallback)}</p>
                <p className="mt-1 text-sm">{previewMutation.data.fallbackAllowed ? t(strings.states.allowed) : t(strings.states.blocked)}</p>
              </div>
            </div>
          ) : (
            <p className="mt-4 text-sm text-app-muted-foreground">{t(strings.pages.routePreview.empty)}</p>
          )}

          {selected ? (
            <div className="mt-4 rounded-control border border-app-border p-3">
              <div className="flex flex-wrap items-center gap-2">
                <StatusChip tone="success">{t(strings.pages.routePreview.selected)}</StatusChip>
                <span className="font-mono text-sm">{selected.provider}</span>
                <span className="text-sm text-app-muted-foreground">{selected.locality}</span>
              </div>
              <ul className="mt-2 list-disc pl-5 text-sm text-app-muted-foreground">
                {selected.reasons.map((reason) => (
                  <li key={reason}>{reason}</li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>

        {previewMutation.data ? (
          <div data-testid={selectors.routePreview.candidates} className="overflow-hidden rounded-panel border border-app-border bg-app-surface">
            <table className="w-full min-w-[680px] text-left text-sm">
              <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="px-4 py-3">{t(strings.pages.routePreview.columns.provider)}</th>
                  <th className="px-4 py-3">{t(strings.pages.routePreview.columns.role)}</th>
                  <th className="px-4 py-3">{t(strings.pages.routePreview.columns.locality)}</th>
                  <th className="px-4 py-3">{t(strings.pages.routePreview.columns.status)}</th>
                  <th className="px-4 py-3">{t(strings.pages.routePreview.columns.reason)}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-app-border">
                {previewMutation.data.candidates.map((candidate) => (
                  <tr key={`${candidate.provider}-${candidate.role}-${candidate.locality}`}>
                    <td className="px-4 py-3 font-mono text-xs">{candidate.provider}</td>
                    <td className="px-4 py-3 font-mono text-xs">{candidate.role}</td>
                    <td className="px-4 py-3">{candidate.locality}</td>
                    <td className="px-4 py-3">
                      <StatusChip tone={candidate.selected ? "success" : candidate.fallbackEligible ? "info" : "warning"}>
                        {candidate.selected ? t(strings.pages.routePreview.selected) : candidate.fallbackEligible ? t(strings.pages.routePreview.fallbackCandidate) : t(strings.pages.routePreview.rejected)}
                      </StatusChip>
                    </td>
                    <td className="px-4 py-3 text-app-muted-foreground">{candidate.reasons.join("; ") || t(strings.states.none)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>
    </section>
  );
}

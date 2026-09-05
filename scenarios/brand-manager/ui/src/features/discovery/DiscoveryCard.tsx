import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { DiscoveryResult } from "@vrooli/proto-types/brand-manager/v1/discovery/discovery_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { discoverScenario } from "../../api/discovery";
import { errorMessage } from "../../lib/errorMessage";

/**
 * DiscoveryCard scans a scenario for existing branding state. It is a read
 * surface: the mutating import (which creates a brand) is a CLI/wizard action.
 * The card lets a user enter a scenario name and see what the scanner found —
 * sources, the inferred draft brand, an overall confidence, and suggestions for
 * missing data — before importing from the CLI. Mirrors ApplyCard's shape.
 */
export function DiscoveryCard() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("");
  const [result, setResult] = useState<DiscoveryResult | null>(null);

  const scanMutation = useMutation({
    mutationFn: () => discoverScenario({ scenarioName: scenario }),
    onSuccess: (data) => setResult(data),
  });

  const canScan = scenario.trim().length > 0 && !scanMutation.isPending;

  return (
    <section
      data-testid={selectors.discovery.card}
      aria-label={t(strings.discovery.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.discovery.title)}</h2>
      <p className="mt-1 text-xs text-slate-500">{t(strings.discovery.description)}</p>

      <div className="mt-3 flex flex-col gap-2 sm:flex-row">
        <Input
          data-testid={selectors.discovery.scenarioInput}
          aria-label={t(strings.discovery.scenarioPlaceholder)}
          placeholder={t(strings.discovery.scenarioPlaceholder)}
          value={scenario}
          onChange={(e) => setScenario(e.target.value)}
        />
        <Button
          data-testid={selectors.discovery.scanButton}
          variant="outline"
          onClick={() => scanMutation.mutate()}
          disabled={!canScan}
        >
          {scanMutation.isPending ? t(strings.discovery.scanning) : t(strings.discovery.scanButton)}
        </Button>
      </div>

      {scanMutation.error && (
        <p data-testid={selectors.discovery.error} className="mt-2 text-red-400">
          {errorMessage(scanMutation.error, t)}
        </p>
      )}

      {result && (
        <div data-testid={selectors.discovery.results} className="mt-3 rounded-lg border border-white/10 p-3">
          <p data-testid={selectors.discovery.summary} className="text-xs text-slate-400">
            {t(strings.discovery.scanFor)}{" "}
            <span className="font-medium text-slate-200">{result.scenario}</span>{" "}
            <span className="text-slate-500">
              {`${t(strings.discovery.confidenceLabel)}: ${Math.round(result.confidence * 100)}%`}
            </span>
          </p>

          {result.sources.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-slate-500">{t(strings.discovery.sourcesHeading)}</p>
              <ul data-testid={selectors.discovery.sourcesList} className="mt-1 space-y-1 text-sm text-slate-200">
                {result.sources.map((source) => (
                  <li key={`${source.type}:${source.file}`} className="flex items-center justify-between gap-2">
                    <span className="text-slate-300">{source.file}</span>
                    <span className="text-xs text-slate-500">
                      {`${source.type} · ${Math.round(source.confidence * 100)}%`}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.draftBrand && (
            <div data-testid={selectors.discovery.draft} className="mt-2">
              <p className="text-xs text-slate-500">{t(strings.discovery.draftHeading)}</p>
              <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-slate-200">
                {result.draftBrand.identity?.displayName && (
                  <span>
                    {`${t(strings.discovery.displayNameLabel)}: `}
                    <span className="font-medium">{result.draftBrand.identity.displayName}</span>
                  </span>
                )}
                {result.draftBrand.colors?.primary && (
                  <span className="flex items-center gap-1 text-xs text-slate-400">
                    {t(strings.discovery.primaryLabel)}
                    <span
                      aria-hidden="true"
                      className="inline-block h-3 w-3 rounded-full border border-white/20"
                      style={{ backgroundColor: result.draftBrand.colors.primary }}
                    />
                    {result.draftBrand.colors.primary}
                  </span>
                )}
              </div>
            </div>
          )}

          {result.suggestions.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-slate-500">{t(strings.discovery.suggestionsHeading)}</p>
              <ul data-testid={selectors.discovery.suggestionsList} className="mt-1 space-y-1 text-sm text-slate-400">
                {result.suggestions.map((suggestion) => (
                  <li key={suggestion}>{suggestion}</li>
                ))}
              </ul>
            </div>
          )}

          {result.sources.length === 0 && (
            <p data-testid={selectors.discovery.empty} className="mt-2 text-sm text-slate-500">
              {t(strings.discovery.empty)}
            </p>
          )}
        </div>
      )}
    </section>
  );
}

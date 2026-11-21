import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { ApplyResponse } from "@vrooli/proto-types/brand-manager/v1/apply/apply_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { previewApply } from "../../api/apply";
import { errorMessage } from "../../lib/errorMessage";

/**
 * ApplyCard previews which files applying a brand to a scenario would write. It
 * is a read surface: the mutating apply (which writes into another scenario's
 * source tree) is a CLI/wizard action. The card lets a user enter a brand id +
 * scenario and see the plan — files to write and skipped elements — before
 * committing from the CLI. Mirrors the canonical card structure but adds the two
 * inputs the preview needs.
 */
export function ApplyCard() {
  const { t } = useTranslation();
  const [brandId, setBrandId] = useState("");
  const [scenario, setScenario] = useState("");
  const [preview, setPreview] = useState<ApplyResponse | null>(null);

  const previewMutation = useMutation({
    mutationFn: () => previewApply({ brandId, scenarioName: scenario }),
    onSuccess: (data) => setPreview(data),
  });

  const canPreview = brandId.trim().length > 0 && scenario.trim().length > 0 && !previewMutation.isPending;

  return (
    <section
      data-testid={selectors.apply.card}
      aria-label={t(strings.apply.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.apply.title)}</h2>
      <p className="mt-1 text-xs text-slate-500">{t(strings.apply.description)}</p>

      <div className="mt-3 flex flex-col gap-2 sm:flex-row">
        <Input
          data-testid={selectors.apply.brandInput}
          aria-label={t(strings.apply.brandPlaceholder)}
          placeholder={t(strings.apply.brandPlaceholder)}
          value={brandId}
          onChange={(e) => setBrandId(e.target.value)}
        />
        <Input
          data-testid={selectors.apply.scenarioInput}
          aria-label={t(strings.apply.scenarioPlaceholder)}
          placeholder={t(strings.apply.scenarioPlaceholder)}
          value={scenario}
          onChange={(e) => setScenario(e.target.value)}
        />
        <Button
          data-testid={selectors.apply.previewButton}
          variant="outline"
          onClick={() => previewMutation.mutate()}
          disabled={!canPreview}
        >
          {previewMutation.isPending ? t(strings.apply.previewing) : t(strings.apply.previewButton)}
        </Button>
      </div>

      {previewMutation.error && (
        <p data-testid={selectors.apply.error} className="mt-2 text-red-400">
          {errorMessage(previewMutation.error, t)}
        </p>
      )}

      {preview && (
        <div data-testid={selectors.apply.results} className="mt-3 rounded-lg border border-white/10 p-3">
          <p data-testid={selectors.apply.summary} className="text-xs text-slate-400">
            {t(strings.apply.previewFor)}{" "}
            <span className="font-medium text-slate-200">{preview.scenario}</span>{" "}
            <span className="text-slate-500">{`v${preview.brandVersion}`}</span>
          </p>

          {preview.applied.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-slate-500">{t(strings.apply.appliedHeading)}</p>
              <ul data-testid={selectors.apply.appliedList} className="mt-1 space-y-1 text-sm text-slate-200">
                {preview.applied.map((action) => (
                  <li key={`${action.element}:${action.file}`} className="flex items-center justify-between gap-2">
                    <span className="text-slate-300">{action.file}</span>
                    <span className="text-xs text-slate-500">{`${action.element} · ${action.type}`}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {preview.skipped.length > 0 && (
            <div className="mt-2">
              <p className="text-xs text-slate-500">{t(strings.apply.skippedHeading)}</p>
              <ul data-testid={selectors.apply.skippedList} className="mt-1 space-y-1 text-sm text-slate-400">
                {preview.skipped.map((skip) => (
                  <li key={skip.element} className="flex items-center justify-between gap-2">
                    <span>{skip.element}</span>
                    <span className="text-xs text-slate-600">{skip.reason}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {preview.applied.length === 0 && preview.skipped.length === 0 && (
            <p data-testid={selectors.apply.empty} className="mt-2 text-sm text-slate-500">
              {t(strings.apply.empty)}
            </p>
          )}
        </div>
      )}
    </section>
  );
}

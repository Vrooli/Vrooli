import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { GenerateDesignLanguageResponse } from "@vrooli/proto-types/brand-manager/v1/design/design_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { generateDesignLanguage } from "../../api/design";
import { errorMessage } from "../../lib/errorMessage";

/**
 * DesignCard renders a brand as a canonical DESIGN.md document. It is a read
 * surface: the server reads the brand and returns markdown, writing nothing. A
 * user enters a brand id, clicks Generate, and sees the rendered document — copy
 * or pipe it into a scenario's docs from the CLI. Mirrors DiscoveryCard's shape.
 */
export function DesignCard() {
  const { t } = useTranslation();
  const [brandId, setBrandId] = useState("");
  const [result, setResult] = useState<GenerateDesignLanguageResponse | null>(null);

  const generateMutation = useMutation({
    mutationFn: () => generateDesignLanguage({ brandId }),
    onSuccess: (data) => setResult(data),
  });

  const canGenerate = brandId.trim().length > 0 && !generateMutation.isPending;

  return (
    <section
      data-testid={selectors.design.card}
      aria-label={t(strings.design.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.design.title)}</h2>
      <p className="mt-1 text-xs text-slate-500">{t(strings.design.description)}</p>

      <div className="mt-3 flex flex-col gap-2 sm:flex-row">
        <Input
          data-testid={selectors.design.brandInput}
          aria-label={t(strings.design.brandPlaceholder)}
          placeholder={t(strings.design.brandPlaceholder)}
          value={brandId}
          onChange={(e) => setBrandId(e.target.value)}
        />
        <Button
          data-testid={selectors.design.generateButton}
          variant="outline"
          onClick={() => generateMutation.mutate()}
          disabled={!canGenerate}
        >
          {generateMutation.isPending ? t(strings.design.generating) : t(strings.design.generateButton)}
        </Button>
      </div>

      {generateMutation.error && (
        <p data-testid={selectors.design.error} className="mt-2 text-red-400">
          {errorMessage(generateMutation.error, t)}
        </p>
      )}

      {result && (
        <div data-testid={selectors.design.result} className="mt-3 rounded-lg border border-white/10 p-3">
          <p className="text-xs text-slate-500">
            {t(strings.design.resultHeading)}{" "}
            <span className="font-medium text-slate-200">{result.brandId}</span>
          </p>
          <pre
            data-testid={selectors.design.markdown}
            className="mt-2 max-h-96 overflow-auto whitespace-pre-wrap rounded bg-black/30 p-3 text-xs text-slate-200"
          >
            {result.markdown}
          </pre>
        </div>
      )}
    </section>
  );
}

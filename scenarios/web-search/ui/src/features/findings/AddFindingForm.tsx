import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";

import { findingsClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { FindingSource } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

interface CitationRow {
  url: string;
  title: string;
}

/**
 * AddFindingForm is the manual capture path: a required claim, optional
 * confidence + query, and zero-or-more citation (url|title) rows. On success it
 * resets and invalidates the findings list so the new finding appears.
 */
export function AddFindingForm({ findingsKey }: { findingsKey: readonly unknown[] }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [claim, setClaim] = useState("");
  const [confidence, setConfidence] = useState("0.5");
  const [query, setQuery] = useState("");
  const [citations, setCitations] = useState<CitationRow[]>([]);
  const [validationError, setValidationError] = useState(false);

  const mutation = useMutation({
    mutationFn: async () =>
      findingsClient.addFinding({
        claim: claim.trim(),
        confidence: Number.parseFloat(confidence) || 0,
        query: query.trim(),
        source: FindingSource.MANUAL,
        briefId: "",
        citations: citations
          .filter((c) => c.url.trim() !== "")
          .map((c) => ({ url: c.url.trim(), title: c.title.trim() })),
      }),
    onSuccess: () => {
      setClaim("");
      setConfidence("0.5");
      setQuery("");
      setCitations([]);
      void queryClient.invalidateQueries({ queryKey: findingsKey });
    },
  });

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!claim.trim()) {
      setValidationError(true);
      return;
    }
    setValidationError(false);
    mutation.mutate();
  };

  const updateCitation = (index: number, patch: Partial<CitationRow>) => {
    setCitations((rows) => rows.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  };

  return (
    <form
      data-testid={selectors.findings.addForm}
      onSubmit={handleSubmit}
      aria-labelledby="add-finding-heading"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="add-finding-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.findings.addHeading)}
      </h3>

      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.findings.addClaimLabel)}</span>
        <Input
          data-testid={selectors.findings.addClaim}
          value={claim}
          onChange={(e) => setClaim(e.target.value)}
          placeholder={t(strings.findings.addClaimPlaceholder)}
        />
      </label>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-app-muted-foreground">{t(strings.findings.addConfidenceLabel)}</span>
          <Input
            data-testid={selectors.findings.addConfidence}
            type="number"
            min={0}
            max={1}
            step={0.05}
            value={confidence}
            onChange={(e) => setConfidence(e.target.value)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="text-app-muted-foreground">{t(strings.findings.addQueryLabel)}</span>
          <Input
            data-testid={selectors.findings.addQuery}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t(strings.findings.addQueryPlaceholder)}
          />
        </label>
      </div>

      <div className="flex flex-col gap-2">
        <span className="text-sm text-app-muted-foreground">{t(strings.findings.citationsLabel)}</span>
        {citations.map((c, i) => (
          <div key={i} data-testid={selectors.findings.addCitationRow} className="flex gap-2">
            <Input
              data-testid={selectors.findings.addCitationUrl}
              value={c.url}
              onChange={(e) => updateCitation(i, { url: e.target.value })}
              placeholder={t(strings.findings.addCitationUrlPlaceholder)}
            />
            <Input
              data-testid={selectors.findings.addCitationTitle}
              value={c.title}
              onChange={(e) => updateCitation(i, { title: e.target.value })}
              placeholder={t(strings.findings.addCitationTitlePlaceholder)}
            />
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={t(strings.findings.addCitationRemove)}
              onClick={() => setCitations((rows) => rows.filter((_, idx) => idx !== i))}
            >
              <X aria-hidden="true" className="h-4 w-4" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="self-start"
          onClick={() => setCitations((rows) => [...rows, { url: "", title: "" }])}
        >
          {t(strings.findings.addCitationRow)}
        </Button>
      </div>

      {validationError && (
        <p className="text-sm text-app-danger">{t(strings.findings.claimRequired)}</p>
      )}
      {mutation.error != null && (
        <p className="text-sm text-app-danger">
          {t(strings.findings.addError, { message: errorMessage(mutation.error, t) })}
        </p>
      )}

      <Button
        data-testid={selectors.findings.addSubmit}
        type="submit"
        size="sm"
        className="self-start"
        disabled={mutation.isPending}
      >
        {t(strings.findings.addSubmit)}
      </Button>
    </form>
  );
}

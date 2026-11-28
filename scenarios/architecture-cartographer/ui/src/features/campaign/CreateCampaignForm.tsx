import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Textarea } from "@vrooli/react-component-library/Textarea/1.1.1";
import { ErrorState } from "../../components/ErrorState";
import { useCreateCampaign } from "./controllers/useCampaignController";
import { AuditReportParseError, parseAuditReport } from "./lib/parseAuditReport";
import type { ArchitectureFinding } from "@vrooli/proto-types/architecture/v1/findings_pb";
import { cn } from "../../lib/utils";

export interface CreateCampaignFormProps {
  scenario: string;
  /** Called with the new campaign id once creation succeeds. */
  onCreated: (campaignId: string) => void;
  onCancel: () => void;
}

interface ParseState {
  findings: ArchitectureFinding[];
  error: string | null;
}

/**
 * Seed a campaign by pasting a test-genie `--json` report — the browser twin
 * of the CLI's `campaign create --from-audit`. The findings are parsed
 * client-side (no server-side proto-JSON fragility) and ingested via
 * CreateCampaign; the tracker recomputes each afid, so a campaign seeded
 * here reconciles cleanly against a future re-audit from the same report.
 */
export function CreateCampaignForm({ scenario, onCreated, onCancel }: CreateCampaignFormProps) {
  const { t } = useTranslation();
  const [name, setName] = React.useState("");
  const [report, setReport] = React.useState("");
  const [parse, setParse] = React.useState<ParseState>({ findings: [], error: null });
  const create = useCreateCampaign(scenario);

  const onReportChange = (value: string) => {
    setReport(value);
    if (value.trim().length === 0) {
      setParse({ findings: [], error: null });
      return;
    }
    try {
      setParse({ findings: parseAuditReport(value), error: null });
    } catch (err) {
      const message = err instanceof AuditReportParseError ? err.message : String(err);
      setParse({ findings: [], error: message });
    }
  };

  const canSubmit = report.trim().length > 0 && parse.error === null && !create.isPending;

  const onSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!canSubmit) return;
    create.mutate(
      { name: name.trim(), findings: parse.findings },
      {
        onSuccess: (resp) => {
          const id = resp.status?.campaign?.id;
          if (id) onCreated(id);
        },
      },
    );
  };

  return (
    <form
      data-testid={selectors.features.campaign.create.root}
      onSubmit={onSubmit}
      className="flex flex-col gap-3 rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <h4 className="text-sm font-semibold">{t(strings.pages.campaign.create.heading)}</h4>

      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.pages.campaign.create.nameLabel)}</span>
        <Input
          data-testid={selectors.features.campaign.create.nameInput}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t(strings.pages.campaign.create.namePlaceholder)}
        />
      </label>

      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.pages.campaign.create.reportLabel)}</span>
        <Textarea
          data-testid={selectors.features.campaign.create.reportInput}
          value={report}
          onChange={(e) => onReportChange(e.target.value)}
          placeholder={t(strings.pages.campaign.create.reportPlaceholder)}
          rows={6}
          className="font-mono text-xs"
        />
      </label>

      <p className="text-xs text-app-muted-foreground">
        {t(strings.pages.campaign.create.hint, { scenario })}
      </p>

      {parse.error ? (
        <div data-testid={selectors.features.campaign.create.parseError}>
          <ErrorState title={t(strings.pages.campaign.create.parseErrorTitle)} message={parse.error} />
        </div>
      ) : report.trim().length > 0 ? (
        <p
          data-testid={selectors.features.campaign.create.parsedCount}
          className="text-xs text-app-success"
        >
          {t(strings.pages.campaign.create.parsedCount, { count: parse.findings.length })}
        </p>
      ) : null}

      {create.isError ? (
        <ErrorState
          title={t(strings.pages.campaign.errorTitle)}
          message={create.error instanceof Error ? create.error.message : String(create.error)}
        />
      ) : null}

      <div className="flex flex-wrap gap-2">
        <Button
          type="submit"
          variant="primary"
          size="sm"
          data-testid={selectors.features.campaign.create.submit}
          disabled={!canSubmit}
          shape="square"
        >
          {create.isPending
            ? t(strings.pages.campaign.create.submitting)
            : t(strings.pages.campaign.create.submit)}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          data-testid={selectors.features.campaign.create.cancel}
          onClick={onCancel}
          disabled={create.isPending}
          shape="square"
          className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
        >
          {t(strings.pages.campaign.create.cancel)}
        </Button>
      </div>
    </form>
  );
}

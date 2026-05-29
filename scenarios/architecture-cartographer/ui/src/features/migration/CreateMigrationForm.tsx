import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { ErrorState } from "../../components/ErrorState";
import { useCreateMigration } from "./controllers/useMigrationController";
import { AuditReportParseError, parseAuditReport } from "./lib/parseAuditReport";
import type { ArchitectureFinding } from "@vrooli/proto-types/architecture/v1/findings_pb";

export interface CreateMigrationFormProps {
  scenario: string;
  /** Called with the new migration id once creation succeeds. */
  onCreated: (migrationId: string) => void;
  onCancel: () => void;
}

interface ParseState {
  findings: ArchitectureFinding[];
  error: string | null;
}

/**
 * Seed a migration by pasting a test-genie `--json` report — the browser twin
 * of the CLI's `migration create --from-audit`. The findings are parsed
 * client-side (no server-side proto-JSON fragility) and ingested via
 * CreateMigration; the tracker recomputes each afid, so a migration seeded
 * here reconciles cleanly against a future re-audit from the same report.
 */
export function CreateMigrationForm({ scenario, onCreated, onCancel }: CreateMigrationFormProps) {
  const { t } = useTranslation();
  const [name, setName] = React.useState("");
  const [report, setReport] = React.useState("");
  const [parse, setParse] = React.useState<ParseState>({ findings: [], error: null });
  const create = useCreateMigration(scenario);

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
          const id = resp.status?.migration?.id;
          if (id) onCreated(id);
        },
      },
    );
  };

  return (
    <form
      data-testid={selectors.features.migration.create.root}
      onSubmit={onSubmit}
      className="flex flex-col gap-3 rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <h4 className="text-sm font-semibold">{t(strings.pages.migration.create.heading)}</h4>

      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.pages.migration.create.nameLabel)}</span>
        <Input
          data-testid={selectors.features.migration.create.nameInput}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t(strings.pages.migration.create.namePlaceholder)}
        />
      </label>

      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.pages.migration.create.reportLabel)}</span>
        <Textarea
          data-testid={selectors.features.migration.create.reportInput}
          value={report}
          onChange={(e) => onReportChange(e.target.value)}
          placeholder={t(strings.pages.migration.create.reportPlaceholder)}
          rows={6}
          className="font-mono text-xs"
        />
      </label>

      <p className="text-xs text-app-muted-foreground">
        {t(strings.pages.migration.create.hint, { scenario })}
      </p>

      {parse.error ? (
        <div data-testid={selectors.features.migration.create.parseError}>
          <ErrorState title={t(strings.pages.migration.create.parseErrorTitle)} message={parse.error} />
        </div>
      ) : report.trim().length > 0 ? (
        <p
          data-testid={selectors.features.migration.create.parsedCount}
          className="text-xs text-app-success"
        >
          {t(strings.pages.migration.create.parsedCount, { count: parse.findings.length })}
        </p>
      ) : null}

      {create.isError ? (
        <ErrorState
          title={t(strings.pages.migration.errorTitle)}
          message={create.error instanceof Error ? create.error.message : String(create.error)}
        />
      ) : null}

      <div className="flex flex-wrap gap-2">
        <Button
          type="submit"
          variant="default"
          size="sm"
          data-testid={selectors.features.migration.create.submit}
          disabled={!canSubmit}
        >
          {create.isPending
            ? t(strings.pages.migration.create.submitting)
            : t(strings.pages.migration.create.submit)}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid={selectors.features.migration.create.cancel}
          onClick={onCancel}
          disabled={create.isPending}
        >
          {t(strings.pages.migration.create.cancel)}
        </Button>
      </div>
    </form>
  );
}

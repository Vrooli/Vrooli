import { useState, type FormEvent } from "react";
import { X } from "lucide-react";
import type { MatrixRow } from "@vrooli/proto-types/business-health/v1/contract/contract_pb";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { StatusChip } from "../../components/StatusChip";
import { EvidenceSummary } from "./EvidenceSummary";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useLogAttestation } from "./useMatrix";
import { useTranslation } from "../../i18n";

export interface RequirementDrawerProps {
  readonly scenario: string;
  readonly row: MatrixRow;
  readonly onClose: () => void;
}

/**
 * Requirement detail panel: declared validations (with ref-resolution markers),
 * the evidence rollup, and a manual-attestation form. Renders as a right-hand
 * inspector on desktop and a full-screen sheet on mobile (DESIGN.md dialog
 * transformation).
 */
export function RequirementDrawer({ scenario, row, onClose }: RequirementDrawerProps) {
  const { t } = useTranslation();
  const [attestedBy, setAttestedBy] = useState("");
  const [notes, setNotes] = useState("");
  const mutation = useLogAttestation(scenario);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (!attestedBy.trim()) return;
    mutation.mutate(
      { requirementId: row.requirementId, attestedBy: attestedBy.trim(), notes: notes.trim() },
      {
        onSuccess: () => {
          setNotes("");
        },
      },
    );
  };

  return (
    <div
      role="dialog"
      data-testid={selectors.matrix.drawer}
      aria-label={t(strings.matrix.drawer.title)}
      className="flex h-full w-full flex-col overflow-auto border-app-border bg-app-surface md:w-96 md:border-l"
    >
      <header className="flex items-start justify-between gap-3 border-b border-app-border px-4 py-3">
        <div className="min-w-0">
          <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {row.requirementId || row.otId}
          </p>
          <h3 data-testid={selectors.matrix.drawerTitle} className="text-sm font-semibold text-app-foreground">
            {row.requirementTitle || row.otTitle || t(strings.matrix.drawer.title)}
          </h3>
        </div>
        <button
          type="button"
          data-testid={selectors.matrix.drawerClose}
          onClick={onClose}
          aria-label={t(strings.common.close)}
          className="rounded-control p-1 text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
        >
          <X aria-hidden="true" className="h-4 w-4" />
        </button>
      </header>

      <div className="flex flex-col gap-4 px-4 py-4">
        {row.unproven && (
          <StatusChip tone="danger">{t(strings.matrix.unproven)}</StatusChip>
        )}
        {row.unproven && row.unprovenReason && (
          <p className="text-xs text-app-danger">{row.unprovenReason}</p>
        )}

        <section className="flex flex-col gap-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
            {t(strings.matrix.drawer.validations)}
          </h4>
          {row.validations.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.matrix.drawer.noValidations)}
            </p>
          ) : (
            <ul className="flex flex-col gap-2">
              {row.validations.map((validation, index) => (
                <li
                  key={`${validation.type}-${validation.phase}-${index}`}
                  className="rounded-control border border-app-border bg-app-surface-muted p-2 text-xs"
                >
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className="font-medium text-app-foreground">{validation.type}</span>
                    <span className="text-app-muted-foreground">·</span>
                    <span className="text-app-muted-foreground">{validation.phase}</span>
                    <span className="text-app-muted-foreground">·</span>
                    <span className="text-app-muted-foreground">{validation.status}</span>
                  </div>
                  {validation.ref && (
                    <div className="mt-1 flex flex-wrap items-center gap-2">
                      <code className="min-w-0 truncate text-app-foreground">{validation.ref}</code>
                      <StatusChip tone={validation.refExists ? "success" : "danger"}>
                        {validation.refExists
                          ? t(strings.matrix.drawer.refExists)
                          : t(strings.matrix.drawer.refMissing)}
                      </StatusChip>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>

        <section className="flex flex-col gap-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
            {t(strings.matrix.drawer.evidenceHeading)}
          </h4>
          <div data-testid={selectors.matrix.evidenceDetail}>
            <EvidenceSummary evidence={row.evidence} verbose />
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
            {t(strings.matrix.drawer.attestHeading)}
          </h4>
          <form
            data-testid={selectors.matrix.attestForm}
            onSubmit={submit}
            className="flex flex-col gap-2"
          >
            <label className="flex flex-col gap-1 text-xs">
              <span className="text-app-muted-foreground">{t(strings.matrix.drawer.attestBy)}</span>
              <Input
                data-testid={selectors.matrix.attestBy}
                value={attestedBy}
                onChange={(event) => setAttestedBy(event.target.value)}
                placeholder={t(strings.matrix.drawer.attestByPlaceholder)}
                className="border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs">
              <span className="text-app-muted-foreground">{t(strings.matrix.drawer.attestNotes)}</span>
              <Textarea
                data-testid={selectors.matrix.attestNotes}
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
                placeholder={t(strings.matrix.drawer.attestNotesPlaceholder)}
                className="min-h-[64px] border-app-border bg-app-surface text-app-foreground placeholder:text-app-muted-foreground focus:ring-app-focus"
              />
            </label>
            <Button
              data-testid={selectors.matrix.attestSubmit}
              type="submit"
              disabled={mutation.isPending || !attestedBy.trim()}
            >
              {t(strings.matrix.drawer.attestSubmit)}
            </Button>
            {mutation.isSuccess && (
              <p className="text-xs text-app-success">{t(strings.matrix.drawer.attestSuccess)}</p>
            )}
            {mutation.isError && (
              <p data-testid={selectors.matrix.attestError} className="text-xs text-app-danger">
                {errorMessage(mutation.error, t)}
              </p>
            )}
          </form>
        </section>
      </div>
    </div>
  );
}

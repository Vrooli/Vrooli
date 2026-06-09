import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { findingsClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { FindingStatus, type Finding } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { StatusBadge } from "./StatusBadge";

type ActiveForm = "none" | "edit" | "supersede" | "flag";

/**
 * FindingCard renders one finding and its management actions: inline edit
 * (claim + confidence), supersede (replacement id + reason), and flag (reason).
 * Each action mutates through FindingsService and invalidates the list query so
 * the card reflects its new state.
 */
export function FindingCard({
  finding,
  findingsKey,
}: {
  finding: Finding;
  findingsKey: readonly unknown[];
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<ActiveForm>("none");

  const invalidate = () => {
    setForm("none");
    void queryClient.invalidateQueries({ queryKey: findingsKey });
  };

  const editMutation = useMutation({
    mutationFn: async (vars: { claim: string; confidence: number }) =>
      findingsClient.editFinding({ id: finding.id, claim: vars.claim, confidence: vars.confidence }),
    onSuccess: invalidate,
  });
  const supersedeMutation = useMutation({
    mutationFn: async (vars: { replacement: string; reason: string }) =>
      findingsClient.supersedeFinding({
        id: finding.id,
        replacement: vars.replacement,
        reason: vars.reason,
      }),
    onSuccess: invalidate,
  });
  const flagMutation = useMutation({
    mutationFn: async (vars: { reason: string }) =>
      findingsClient.flagFinding({ id: finding.id, reason: vars.reason }),
    onSuccess: invalidate,
  });

  return (
    <li
      data-testid={selectors.findings.item}
      data-finding-id={finding.id}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium text-app-foreground">{finding.claim}</p>
        <StatusBadge status={finding.status} />
      </div>

      <p className="text-xs text-app-muted-foreground">
        {t(strings.findings.confidenceLabel, { value: finding.confidence.toFixed(2) })}
        {finding.query && <span className="ms-3">{t(strings.findings.queryLabel, { query: finding.query })}</span>}
      </p>

      {finding.status === FindingStatus.DISPUTED && finding.disputeNote && (
        <p className="text-xs text-app-warning">
          {t(strings.findings.disputeNoteLabel, { note: finding.disputeNote })}
        </p>
      )}
      {finding.status === FindingStatus.SUPERSEDED && finding.supersededBy && (
        <p className="text-xs text-app-muted-foreground">
          {t(strings.findings.supersededByLabel, { id: finding.supersededBy })}
        </p>
      )}

      {finding.citations.length > 0 && (
        <ul className="flex flex-col gap-0.5 text-xs">
          {finding.citations.map((c, i) => (
            <li key={c.id || `${c.url}/${i}`}>
              <a
                href={c.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-app-primary hover:underline"
              >
                {c.title || c.url}
              </a>
            </li>
          ))}
        </ul>
      )}

      {form === "none" && (
        <div className="mt-1 flex flex-wrap gap-2">
          <Button
            data-testid={selectors.findings.editButton}
            variant="outline"
            size="sm"
            onClick={() => setForm("edit")}
          >
            {t(strings.findings.editAction)}
          </Button>
          <Button
            data-testid={selectors.findings.supersedeButton}
            variant="outline"
            size="sm"
            onClick={() => setForm("supersede")}
          >
            {t(strings.findings.supersedeAction)}
          </Button>
          <Button
            data-testid={selectors.findings.flagButton}
            variant="outline"
            size="sm"
            onClick={() => setForm("flag")}
          >
            {t(strings.findings.flagAction)}
          </Button>
        </div>
      )}

      {form === "edit" && (
        <EditForm
          finding={finding}
          pending={editMutation.isPending}
          error={editMutation.error}
          onCancel={() => setForm("none")}
          onSubmit={(claim, confidence) => editMutation.mutate({ claim, confidence })}
        />
      )}
      {form === "supersede" && (
        <SupersedeForm
          pending={supersedeMutation.isPending}
          error={supersedeMutation.error}
          onCancel={() => setForm("none")}
          onSubmit={(replacement, reason) => supersedeMutation.mutate({ replacement, reason })}
        />
      )}
      {form === "flag" && (
        <FlagForm
          pending={flagMutation.isPending}
          error={flagMutation.error}
          onCancel={() => setForm("none")}
          onSubmit={(reason) => flagMutation.mutate({ reason })}
        />
      )}
    </li>
  );
}

type ErrorKeyPath =
  | typeof strings.findings.editError
  | typeof strings.findings.supersedeError
  | typeof strings.findings.flagError;

function FormError({ error, keyPath }: { error: unknown; keyPath: ErrorKeyPath }) {
  const { t } = useTranslation();
  if (error == null) return null;
  return <p className="text-xs text-app-danger">{t(keyPath, { message: errorMessage(error, t) })}</p>;
}

function EditForm({
  finding,
  pending,
  error,
  onCancel,
  onSubmit,
}: {
  finding: Finding;
  pending: boolean;
  error: unknown;
  onCancel: () => void;
  onSubmit: (claim: string, confidence: number) => void;
}) {
  const { t } = useTranslation();
  const [claim, setClaim] = useState(finding.claim);
  const [confidence, setConfidence] = useState(finding.confidence.toString());

  return (
    <form
      data-testid={selectors.findings.editForm}
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit(claim.trim(), Number.parseFloat(confidence) || 0);
      }}
      className="mt-1 flex flex-col gap-2"
    >
      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.findings.editClaimLabel)}</span>
        <Input
          data-testid={selectors.findings.editClaim}
          value={claim}
          onChange={(e) => setClaim(e.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1 text-sm">
        <span className="text-app-muted-foreground">{t(strings.findings.editConfidenceLabel)}</span>
        <Input
          data-testid={selectors.findings.editConfidence}
          type="number"
          min={0}
          max={1}
          step={0.05}
          value={confidence}
          onChange={(e) => setConfidence(e.target.value)}
        />
      </label>
      <FormError error={error} keyPath={strings.findings.editError} />
      <div className="flex gap-2">
        <Button data-testid={selectors.findings.editSave} type="submit" size="sm" disabled={pending}>
          {t(strings.findings.editSave)}
        </Button>
        <Button
          data-testid={selectors.findings.editCancel}
          type="button"
          variant="outline"
          size="sm"
          onClick={onCancel}
        >
          {t(strings.findings.editCancel)}
        </Button>
      </div>
    </form>
  );
}

function SupersedeForm({
  pending,
  error,
  onCancel,
  onSubmit,
}: {
  pending: boolean;
  error: unknown;
  onCancel: () => void;
  onSubmit: (replacement: string, reason: string) => void;
}) {
  const { t } = useTranslation();
  const [replacement, setReplacement] = useState("");
  const [reason, setReason] = useState("");

  return (
    <form
      data-testid={selectors.findings.supersedeForm}
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit(replacement.trim(), reason.trim());
      }}
      className="mt-1 flex flex-col gap-2"
    >
      <Input
        value={replacement}
        onChange={(e) => setReplacement(e.target.value)}
        placeholder={t(strings.findings.supersedeReplacementPlaceholder)}
      />
      <Textarea
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        placeholder={t(strings.findings.supersedeReasonPlaceholder)}
      />
      <FormError error={error} keyPath={strings.findings.supersedeError} />
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={pending}>
          {t(strings.findings.supersedeSubmit)}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          {t(strings.findings.cancel)}
        </Button>
      </div>
    </form>
  );
}

function FlagForm({
  pending,
  error,
  onCancel,
  onSubmit,
}: {
  pending: boolean;
  error: unknown;
  onCancel: () => void;
  onSubmit: (reason: string) => void;
}) {
  const { t } = useTranslation();
  const [reason, setReason] = useState("");

  return (
    <form
      data-testid={selectors.findings.flagForm}
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit(reason.trim());
      }}
      className="mt-1 flex flex-col gap-2"
    >
      <Textarea
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        placeholder={t(strings.findings.flagReasonPlaceholder)}
      />
      <FormError error={error} keyPath={strings.findings.flagError} />
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={pending}>
          {t(strings.findings.flagSubmit)}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          {t(strings.findings.cancel)}
        </Button>
      </div>
    </form>
  );
}

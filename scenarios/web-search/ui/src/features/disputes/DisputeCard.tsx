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
import { type Finding } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import { StatusBadge } from "../findings/StatusBadge";

type Resolution = "keep" | "supersede";

/**
 * DisputeCard renders one DISPUTED finding as a conflict card: the contested
 * claim, its dispute note (why sources conflict), its citations, and a resolve
 * action. Resolution "keep" returns the finding to ACTIVE; "supersede" retires
 * it in favor of a replacement id. Both write an audit row server-side and
 * invalidate the queue so the card disappears once resolved.
 */
export function DisputeCard({ finding, queueKey }: { finding: Finding; queueKey: readonly unknown[] }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [resolution, setResolution] = useState<Resolution>("keep");
  const [replacement, setReplacement] = useState("");
  const [reason, setReason] = useState("");

  const resolveMutation = useMutation({
    mutationFn: async () =>
      findingsClient.resolveDispute({
        id: finding.id,
        resolution,
        replacement: resolution === "supersede" ? replacement.trim() : "",
        reason: reason.trim(),
      }),
    onSuccess: () => {
      setOpen(false);
      void queryClient.invalidateQueries({ queryKey: queueKey });
    },
  });

  return (
    <li
      data-testid={selectors.disputes.item}
      data-finding-id={finding.id}
      className="flex flex-col gap-2 rounded-panel border border-app-warning/50 bg-app-warning/5 p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm font-medium text-app-foreground">{finding.claim}</p>
        <StatusBadge status={finding.status} />
      </div>

      {finding.disputeNote && (
        <p className="text-xs text-app-warning" data-testid={selectors.disputes.note}>
          {t(strings.disputes.noteLabel, { note: finding.disputeNote })}
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

      {!open && (
        <div className="mt-1">
          <Button
            data-testid={selectors.disputes.resolveButton}
            variant="outline"
            size="sm"
            onClick={() => setOpen(true)}
          >
            {t(strings.disputes.resolveAction)}
          </Button>
        </div>
      )}

      {open && (
        <form
          data-testid={selectors.disputes.resolveForm}
          onSubmit={(e) => {
            e.preventDefault();
            resolveMutation.mutate();
          }}
          className="mt-1 flex flex-col gap-2"
        >
          <div role="radiogroup" aria-label={t(strings.disputes.resolutionLabel)} className="flex gap-2">
            <label className="flex items-center gap-1 text-sm text-app-foreground">
              <input
                type="radio"
                name={`resolution-${finding.id}`}
                checked={resolution === "keep"}
                onChange={() => setResolution("keep")}
                className="accent-app-primary"
              />
              {t(strings.disputes.resolutionKeep)}
            </label>
            <label className="flex items-center gap-1 text-sm text-app-foreground">
              <input
                type="radio"
                name={`resolution-${finding.id}`}
                checked={resolution === "supersede"}
                onChange={() => setResolution("supersede")}
                className="accent-app-primary"
              />
              {t(strings.disputes.resolutionSupersede)}
            </label>
          </div>

          {resolution === "supersede" && (
            <Input
              data-testid={selectors.disputes.replacement}
              value={replacement}
              onChange={(e) => setReplacement(e.target.value)}
              placeholder={t(strings.disputes.replacementPlaceholder)}
            />
          )}

          <Textarea
            data-testid={selectors.disputes.reason}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder={t(strings.disputes.reasonPlaceholder)}
          />

          {resolveMutation.error != null && (
            <p className="text-xs text-app-danger">
              {t(strings.disputes.resolveError, { message: errorMessage(resolveMutation.error, t) })}
            </p>
          )}

          <div className="flex gap-2">
            <Button
              data-testid={selectors.disputes.resolveSubmit}
              type="submit"
              size="sm"
              disabled={resolveMutation.isPending}
            >
              {t(strings.disputes.resolveSubmit)}
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => setOpen(false)}>
              {t(strings.disputes.cancel)}
            </Button>
          </div>
        </form>
      )}
    </li>
  );
}

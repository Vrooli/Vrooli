import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ReviewFindingDetail } from "./ReviewFindingDetail";
import { ReviewQueuePanel } from "./ReviewQueuePanel";

import {
  DispositionDecision,
  FindingSeverity,
  FindingStatus,
  assignFinding,
  getReviewSummary,
  listFindingsPage,
  recordDisposition,
  type Finding,
} from "../../api/review";
import { useTranslation } from "../../i18n";
import { Kpi } from "../primitives";

const PAGE_SIZE = 25;
const SEVERITY_LABEL: Record<FindingSeverity, string> = {
  [FindingSeverity.UNSPECIFIED]: "Unspecified",
  [FindingSeverity.BLOCKER]: "Blocker",
  [FindingSeverity.HIGH]: "High",
  [FindingSeverity.MEDIUM]: "Medium",
  [FindingSeverity.LOW]: "Low",
  [FindingSeverity.INFO]: "Info",
};

const SEARCH_DEBOUNCE_MS = 250;

/**
 * Review is a server-owned queue. Findings and dispositions come from
 * ReviewService; this page never reconstructs them from coverage rows. The
 * queue is one filtered, paginated list because a capped multi-column board
 * cannot honestly represent a backlog of thousands.
 */
export function ReviewPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [decision, setDecision] = useState<DispositionDecision>(DispositionDecision.FIX);
  const [reason, setReason] = useState("");
  const [actor, setActor] = useState("local");
  const [owner, setOwner] = useState("");
  const [copied, setCopied] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  const [severity, setSeverity] = useState<FindingSeverity>(FindingSeverity.UNSPECIFIED);
  const [status, setStatus] = useState<FindingStatus>(FindingStatus.UNSPECIFIED);
  const [searchText, setSearchText] = useState("");
  const [query, setQuery] = useState("");
  const [offset, setOffset] = useState(0);

  // Debounce the free-text filter without delaying the page reset: changing a
  // filter returns to page 1 immediately, then the request follows the typing.
  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      setQuery(searchText.trim());
    }, SEARCH_DEBOUNCE_MS);
    return () => globalThis.clearTimeout(timer);
  }, [searchText]);

  const findings = useQuery({
    queryKey: ["review", "findings", { severity, status, query, offset }],
    queryFn: () => listFindingsPage({ severity, status, query, offset, limit: PAGE_SIZE }),
    retry: 0,
  });
  const summary = useQuery({
    queryKey: ["review", "summary"],
    queryFn: getReviewSummary,
    retry: 0,
  });

  const rows = useMemo(() => findings.data?.findings ?? [], [findings.data]);
  const total = findings.data?.total ?? 0;
  const selected = rows.find((finding) => finding.id === selectedId) ?? rows[0];
  const selectedKey = selected?.id;

  // A decision belongs to one finding. Clear the draft when the selection
  // changes so an owner or reason cannot be applied to the wrong row.
  useEffect(() => {
    setOwner("");
    setReason("");
    setMessage(null);
  }, [selectedKey]);

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["review", "findings"] }),
      queryClient.invalidateQueries({ queryKey: ["review", "summary"] }),
    ]);
  };

  const disposition = useMutation({
    mutationFn: () =>
      recordDisposition({
        id: selected?.id ?? "",
        decision,
        reason: reason.trim(),
        actor: actor.trim() || "local",
      }),
    onSuccess: async () => {
      setReason("");
      setMessage(t("workbench.review.dispositionSaved", { defaultValue: "Decision recorded." }));
      await refresh();
    },
    onError: (error: unknown) => {
      setMessage(
        error instanceof Error
          ? error.message
          : t("workbench.review.dispositionFailed", {
              defaultValue: "The decision could not be recorded.",
            }),
      );
    },
  });

  const assignment = useMutation({
    mutationFn: () => assignFinding(selected?.id ?? "", owner.trim()),
    onSuccess: async () => {
      setMessage(t("workbench.review.ownerSaved", { defaultValue: "Owner updated." }));
      await refresh();
    },
  });

  const copyEvidence = async (finding: Finding) => {
    try {
      await navigator.clipboard.writeText(
        `${finding.subjectId} · ${finding.gate} · ${SEVERITY_LABEL[finding.severity]}\n${finding.message}\n${finding.evidenceJson}`,
      );
      setCopied(true);
      globalThis.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  };

  const changeSearch = (value: string) => {
    setSearchText(value);
    setOffset(0);
  };

  const counts = summary.data;
  const rangeStart = offset + 1;
  const rangeEnd = offset + rows.length;
  const canGoBack = offset > 0;
  const canGoForward = offset + rows.length < total;

  return (
    <div className="wb-page" data-testid="review-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.review.kicker", { defaultValue: "Library · Review" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.review.title", { defaultValue: "What needs a person" })}
          </h1>
          <p className="wb-lede">
            {t("workbench.review.lede", {
              defaultValue:
                "Server-owned findings with durable decisions. A finding is real evidence, not a browser reconstruction.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <span className="wb-mono" style={{ fontSize: 26, letterSpacing: "-0.03em" }}>
            {counts?.open ?? "—"}
          </span>
          <span className="wb-muted wb-small">
            {t("workbench.review.open", { defaultValue: "open findings" })}
          </span>
        </div>
      </div>

      <div className="wb-kpis">
        <Kpi
          label={t("workbench.review.kpiAccepted", { defaultValue: "Accepted" })}
          value={counts?.accepted ?? "—"}
          detail={t("workbench.review.kpiAcceptedDetail", {
            defaultValue: "Accepted with a reason",
          })}
        />
        <Kpi
          label={t("workbench.review.kpiBlocking", { defaultValue: "Blocking" })}
          value={counts?.blockers ?? "—"}
          tone="red"
          detail={t("workbench.review.kpiBlockingDetail", { defaultValue: "Blocks a release" })}
        />
        <Kpi
          label={t("workbench.review.kpiHigh", { defaultValue: "High" })}
          value={counts?.high ?? "—"}
          tone="amber"
          detail={t("workbench.review.kpiHighDetail", { defaultValue: "Needs a decision soon" })}
        />
        <Kpi
          label={t("workbench.review.kpiAcknowledged", { defaultValue: "Acknowledged" })}
          value={counts?.acknowledged ?? "—"}
          tone="signal"
          detail={t("workbench.review.kpiAcknowledgedDetail", {
            defaultValue: "Known and owned",
          })}
        />
      </div>

      <div className="wb-review-layout">
        <ReviewQueuePanel
          loading={findings.isLoading}
          error={findings.isError}
          rows={rows}
          total={total}
          rangeStart={rangeStart}
          rangeEnd={rangeEnd}
          canGoBack={canGoBack}
          canGoForward={canGoForward}
          severity={severity}
          setSeverity={setSeverity}
          status={status}
          setStatus={setStatus}
          searchText={searchText}
          changeSearch={changeSearch}
          selectedId={selected?.id}
          setSelectedId={setSelectedId}
          setOffset={setOffset}
        />
        {selected ? (
          <ReviewFindingDetail
            selected={selected}
            copied={copied}
            copyEvidence={copyEvidence}
            decision={decision}
            setDecision={setDecision}
            owner={owner}
            setOwner={setOwner}
            assignmentPending={assignment.isPending}
            onAssign={() => assignment.mutate()}
            reason={reason}
            setReason={setReason}
            actor={actor}
            setActor={setActor}
            dispositionPending={disposition.isPending}
            onRecord={() => disposition.mutate()}
            message={message}
          />
        ) : null}
      </div>
    </div>
  );
}

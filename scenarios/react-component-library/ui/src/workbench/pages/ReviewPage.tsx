import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, CircleAlert, Copy, ShieldCheck } from "lucide-react";

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
import {
  Banner,
  Chip,
  Empty,
  Field,
  Kpi,
  Panel,
  PanelBody,
  PanelHead,
  SearchBox,
  Select,
  Skeleton,
  WbButton,
} from "../primitives";

const PAGE_SIZE = 25;
const SEARCH_DEBOUNCE_MS = 250;

const SEVERITY_LABEL: Record<FindingSeverity, string> = {
  [FindingSeverity.UNSPECIFIED]: "Unspecified",
  [FindingSeverity.BLOCKER]: "Blocker",
  [FindingSeverity.HIGH]: "High",
  [FindingSeverity.MEDIUM]: "Medium",
  [FindingSeverity.LOW]: "Low",
  [FindingSeverity.INFO]: "Info",
};

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
  const [decision, setDecision] = useState<DispositionDecision>(
    DispositionDecision.FIX,
  );
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
        <Panel>
          <PanelHead
            title={t("workbench.review.queue", { defaultValue: "Review queue" })}
            meta={t("workbench.review.queueMeta", { defaultValue: "Server-owned, newest first" })}
          />
          <PanelBody>
            <div className="wb-stack">
              <div
                style={{
                  display: "flex",
                  flexWrap: "wrap",
                  gap: 12,
                  alignItems: "flex-end",
                }}
              >
                <Field
                  label={t("workbench.review.severityFilter", { defaultValue: "Severity" })}
                >
                  <Select
                    style={{ minWidth: 150 }}
                    value={severity}
                    onChange={(event) => {
                      setSeverity(Number(event.target.value));
                      setOffset(0);
                    }}
                  >
                    <option value={FindingSeverity.UNSPECIFIED}>
                      {t("workbench.review.allSeverities", { defaultValue: "All severities" })}
                    </option>
                    <option value={FindingSeverity.BLOCKER}>
                      {t("workbench.review.blocking", { defaultValue: "Blocking" })}
                    </option>
                    <option value={FindingSeverity.HIGH}>
                      {t("workbench.review.high", { defaultValue: "High" })}
                    </option>
                    <option value={FindingSeverity.MEDIUM}>
                      {t("workbench.review.severityMedium", { defaultValue: "Medium" })}
                    </option>
                    <option value={FindingSeverity.LOW}>
                      {t("workbench.review.severityLow", { defaultValue: "Low" })}
                    </option>
                    <option value={FindingSeverity.INFO}>
                      {t("workbench.review.severityInfo", { defaultValue: "Info" })}
                    </option>
                  </Select>
                </Field>

                <Field label={t("workbench.review.statusFilter", { defaultValue: "Status" })}>
                  <Select
                    style={{ minWidth: 150 }}
                    value={status}
                    onChange={(event) => {
                      setStatus(Number(event.target.value));
                      setOffset(0);
                    }}
                  >
                    <option value={FindingStatus.UNSPECIFIED}>
                      {t("workbench.review.allStatuses", { defaultValue: "All statuses" })}
                    </option>
                    <option value={FindingStatus.OPEN}>
                      {t("workbench.review.statusOpen", { defaultValue: "Open" })}
                    </option>
                    <option value={FindingStatus.ACKNOWLEDGED}>
                      {t("workbench.review.statusAcknowledged", { defaultValue: "Acknowledged" })}
                    </option>
                    <option value={FindingStatus.ACCEPTED}>
                      {t("workbench.review.statusAccepted", { defaultValue: "Accepted" })}
                    </option>
                    <option value={FindingStatus.FIXED}>
                      {t("workbench.review.statusFixed", { defaultValue: "Fixed" })}
                    </option>
                    <option value={FindingStatus.RETIRED}>
                      {t("workbench.review.statusRetired", { defaultValue: "Retired" })}
                    </option>
                  </Select>
                </Field>

                <div className="wb-field" style={{ flex: "2 1 240px", minWidth: 0 }}>
                  <span>{t("workbench.review.searchLabel", { defaultValue: "Search" })}</span>
                  <SearchBox
                    value={searchText}
                    onChange={changeSearch}
                    placeholder={t("workbench.review.searchPlaceholder", {
                      defaultValue: "Search message, gate or asset",
                    })}
                    label={t("workbench.review.searchLabel", { defaultValue: "Search" })}
                  />
                </div>
              </div>

              {findings.isLoading ? (
                <div className="wb-stack tight" aria-hidden>
                  <Skeleton height={78} />
                  <Skeleton height={78} />
                  <Skeleton height={78} />
                </div>
              ) : findings.isError ? (
                <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
                  <b>
                    {t("workbench.review.unavailableTitle", {
                      defaultValue: "The findings service did not answer.",
                    })}
                  </b>{" "}
                  {t("workbench.review.unavailableBody", {
                    defaultValue:
                      "Review shows unavailable rather than an empty queue. Retry when the API is reachable.",
                  })}
                </Banner>
              ) : total === 0 ? (
                <Empty
                  title={t("workbench.review.emptyTitle", {
                    defaultValue: "Nothing needs a decision",
                  })}
                >
                  {t("workbench.review.emptyBody", {
                    defaultValue:
                      "The findings service returned a measured empty queue. It is not a substitute for a missing report.",
                  })}
                </Empty>
              ) : rows.length === 0 ? (
                <Empty
                  title={t("workbench.review.pageEmptyTitle", {
                    defaultValue: "This page is empty",
                  })}
                  action={
                    <WbButton variant="ghost" size="sm" onClick={() => setOffset(0)}>
                      {t("workbench.review.backToFirst", {
                        defaultValue: "Back to first page",
                      })}
                    </WbButton>
                  }
                >
                  {t("workbench.review.pageEmptyBody", {
                    defaultValue:
                      "The service reports {{total}} findings, but this page has none. Another writer may have changed the queue.",
                    total: total.toLocaleString(),
                  })}
                </Empty>
              ) : (
                <div className="wb-stack tight">
                  {rows.map((finding) => (
                    <button
                      key={finding.id}
                      type="button"
                      className="wb-finding"
                      aria-pressed={finding.id === selected?.id}
                      onClick={() => setSelectedId(finding.id)}
                    >
                      <b>{finding.subjectId}</b>
                      <small>
                        {finding.gate} · {SEVERITY_LABEL[finding.severity]}
                      </small>
                      <p>{finding.message}</p>
                      <span className="wb-finding-meta">
                        <Chip
                          tone={
                            finding.severity === FindingSeverity.BLOCKER
                              ? "red"
                              : finding.severity === FindingSeverity.HIGH
                                ? "amber"
                                : undefined
                          }
                        >
                          {SEVERITY_LABEL[finding.severity]}
                        </Chip>
                        <span className="wb-tag">
                          <StatusLabel status={finding.status} />
                        </span>
                        {finding.owner ? <span className="wb-tag">{finding.owner}</span> : null}
                      </span>
                    </button>
                  ))}
                </div>
              )}

              {!findings.isLoading && !findings.isError && rows.length > 0 ? (
                <div className="wb-row-between" style={{ flexWrap: "wrap", gap: 8 }}>
                  <span className="wb-muted wb-small" data-testid="review-range">
                    {t("workbench.review.range", {
                      defaultValue: "{{start}}–{{end}} of {{total}}",
                      start: rangeStart,
                      end: rangeEnd,
                      total: total.toLocaleString(),
                    })}
                  </span>
                  <div className="wb-row" style={{ gap: 8 }}>
                    <WbButton
                      variant="ghost"
                      size="sm"
                      disabled={!canGoBack}
                      onClick={() => setOffset((current) => Math.max(0, current - PAGE_SIZE))}
                    >
                      {t("workbench.review.previous", { defaultValue: "Previous" })}
                    </WbButton>
                    <WbButton
                      variant="ghost"
                      size="sm"
                      disabled={!canGoForward}
                      onClick={() => setOffset((current) => current + PAGE_SIZE)}
                    >
                      {t("workbench.review.next", { defaultValue: "Next" })}
                    </WbButton>
                  </div>
                </div>
              ) : null}
            </div>
          </PanelBody>
        </Panel>

        {selected ? (
          <Panel
            role="region"
            aria-label={t("workbench.review.detail", { defaultValue: "Finding detail" })}
          >
            <PanelHead
              title={selected.subjectId}
              meta={`${selected.gate} · ${SEVERITY_LABEL[selected.severity]}`}
            />
            <PanelBody>
              <p>{selected.message}</p>
              {selected.remediation ? (
                <p className="wb-muted">
                  <b>{t("workbench.review.remediation", { defaultValue: "Remediation" })}:</b>{" "}
                  {selected.remediation}
                </p>
              ) : null}
              <div className="wb-row" style={{ flexWrap: "wrap", gap: 6 }}>
                <Chip tone={selected.severity === FindingSeverity.BLOCKER ? "red" : "amber"}>
                  {SEVERITY_LABEL[selected.severity]}
                </Chip>
                <span className="wb-tag">
                  <StatusLabel status={selected.status} />
                </span>
                {selected.owner ? <span className="wb-tag">{selected.owner}</span> : null}
                <button
                  type="button"
                  className="wb-btn quiet sm"
                  onClick={() => void copyEvidence(selected)}
                >
                  <Copy aria-hidden size={12} />
                  {copied
                    ? t("workbench.review.copied", { defaultValue: "Copied" })
                    : t("workbench.review.copyEvidence", { defaultValue: "Copy evidence" })}
                </button>
                {selected.docsRef ? (
                  <a className="wb-btn quiet sm" href={`/${selected.docsRef}`} target="_blank" rel="noreferrer">
                    {t("workbench.review.docs", { defaultValue: "Docs" })}
                    <ArrowUpRight aria-hidden size={12} />
                  </a>
                ) : null}
              </div>

              <details className="wb-muted" style={{ marginTop: 10 }}>
                <summary>{t("workbench.review.evidence", { defaultValue: "Evidence" })}</summary>
                <pre style={{ whiteSpace: "pre-wrap", marginTop: 6 }}>{selected.evidenceJson}</pre>
              </details>

              <div style={{ marginTop: 14 }}>
                <span className="wb-caption">
                  {t("workbench.review.history", { defaultValue: "Decision history" })}
                </span>
                {selected.dispositions.length === 0 ? (
                  <p className="wb-muted-2 wb-small">
                    {t("workbench.review.none", { defaultValue: "None" })}
                  </p>
                ) : (
                  selected.dispositions.map((entry) => (
                    <div key={entry.id} className="wb-finding">
                      <b>
                        <DecisionLabel decision={entry.decision} />
                      </b>
                      <small>
                        {entry.actor} · {entry.at}
                      </small>
                      <p>{entry.reason}</p>
                    </div>
                  ))
                )}
              </div>

              <div className="wb-stack" style={{ marginTop: 14 }}>
                <span className="wb-caption">
                  {t("workbench.review.decision", { defaultValue: "Needs a decision" })}
                </span>
                <label className="wb-field">
                  <span>{t("workbench.review.owner", { defaultValue: "Owner" })}</span>
                  <input
                    className="wb-input"
                    value={owner || selected.owner}
                    onChange={(event) => setOwner(event.target.value)}
                    placeholder={t("workbench.review.ownerPlaceholder", { defaultValue: "name@example" })}
                  />
                </label>
                <WbButton
                  variant="ghost"
                  size="sm"
                  disabled={!owner.trim() || assignment.isPending}
                  onClick={() => assignment.mutate()}
                >
                  {t("workbench.review.assign", { defaultValue: "Assign owner" })}
                </WbButton>
                <label className="wb-field">
                  <span>{t("workbench.review.decisionLabel", { defaultValue: "Decision" })}</span>
                  <select
                    className="wb-select"
                    value={decision}
                    onChange={(event) => setDecision(Number(event.target.value))}
                  >
                    <option value={DispositionDecision.FIX}>
                      {t("workbench.review.fix", { defaultValue: "Fix" })}
                    </option>
                    <option value={DispositionDecision.ACKNOWLEDGE}>
                      {t("workbench.review.acknowledge", { defaultValue: "Acknowledge" })}
                    </option>
                    <option value={DispositionDecision.ACCEPT_RISK}>
                      {t("workbench.review.acceptRisk", { defaultValue: "Accept risk" })}
                    </option>
                    <option value={DispositionDecision.RETIRE}>
                      {t("workbench.review.retire", { defaultValue: "Retire" })}
                    </option>
                  </select>
                </label>
                <label className="wb-field">
                  <span>{t("workbench.review.reason", { defaultValue: "Reason" })}</span>
                  <textarea
                    className="wb-textarea"
                    rows={3}
                    value={reason}
                    onChange={(event) => setReason(event.target.value)}
                  />
                </label>
                <label className="wb-field">
                  <span>{t("workbench.review.actor", { defaultValue: "Actor" })}</span>
                  <input
                    className="wb-input"
                    value={actor}
                    onChange={(event) => setActor(event.target.value)}
                  />
                </label>
                <WbButton
                  variant="primary"
                  size="sm"
                  disabled={!reason.trim() || disposition.isPending}
                  onClick={() => disposition.mutate()}
                >
                  <ShieldCheck aria-hidden size={14} />
                  {t("workbench.review.record", { defaultValue: "Record decision" })}
                </WbButton>
                {message ? <p className="wb-muted wb-small" role="status">{message}</p> : null}
              </div>
            </PanelBody>
          </Panel>
        ) : null}
      </div>
    </div>
  );
}

// Inline `t("key", { defaultValue })` calls keep the catalog generator aware
// of every status and decision label; a dynamic key lookup would make the
// strings invisible to the catalog and let them drift out of en.json.
function StatusLabel({ status }: { status: FindingStatus }) {
  const { t } = useTranslation();
  switch (status) {
    case FindingStatus.ACKNOWLEDGED:
      return <>{t("workbench.review.statusAcknowledged", { defaultValue: "Acknowledged" })}</>;
    case FindingStatus.ACCEPTED:
      return <>{t("workbench.review.statusAccepted", { defaultValue: "Accepted" })}</>;
    case FindingStatus.FIXED:
      return <>{t("workbench.review.statusFixed", { defaultValue: "Fixed" })}</>;
    case FindingStatus.RETIRED:
      return <>{t("workbench.review.statusRetired", { defaultValue: "Retired" })}</>;
    default:
      return <>{t("workbench.review.statusOpen", { defaultValue: "Open" })}</>;
  }
}

function DecisionLabel({ decision }: { decision: DispositionDecision }) {
  const { t } = useTranslation();
  switch (decision) {
    case DispositionDecision.ACKNOWLEDGE:
      return <>{t("workbench.review.acknowledge", { defaultValue: "Acknowledge" })}</>;
    case DispositionDecision.ACCEPT_RISK:
      return <>{t("workbench.review.acceptRisk", { defaultValue: "Accept risk" })}</>;
    case DispositionDecision.FIX:
      return <>{t("workbench.review.fix", { defaultValue: "Fix" })}</>;
    case DispositionDecision.RETIRE:
      return <>{t("workbench.review.retire", { defaultValue: "Retire" })}</>;
    default:
      return <>{t("workbench.review.decisionLabel", { defaultValue: "Decision" })}</>;
  }
}

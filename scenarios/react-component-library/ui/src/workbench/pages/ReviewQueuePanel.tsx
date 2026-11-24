import { CircleAlert } from "lucide-react";
import { FindingSeverity, FindingStatus, type Finding } from "../../api/review";
import { useTranslation } from "../../i18n";
import {
  Banner,
  Chip,
  Empty,
  Field,
  Panel,
  PanelBody,
  PanelHead,
  SearchBox,
  Select,
  Skeleton,
  WbButton,
} from "../primitives";

const PAGE_SIZE = 25;
const SEVERITY_LABEL: Record<FindingSeverity, string> = {
  [FindingSeverity.UNSPECIFIED]: "Unspecified",
  [FindingSeverity.BLOCKER]: "Blocker",
  [FindingSeverity.HIGH]: "High",
  [FindingSeverity.MEDIUM]: "Medium",
  [FindingSeverity.LOW]: "Low",
  [FindingSeverity.INFO]: "Info",
};
type Props = {
  loading: boolean;
  error: boolean;
  rows: Finding[];
  total: number;
  rangeStart: number;
  rangeEnd: number;
  canGoBack: boolean;
  canGoForward: boolean;
  severity: FindingSeverity;
  setSeverity: (v: FindingSeverity) => void;
  status: FindingStatus;
  setStatus: (v: FindingStatus) => void;
  searchText: string;
  changeSearch: (v: string) => void;
  selectedId?: string;
  setSelectedId: (id: string) => void;
  setOffset: (value: number | ((current: number) => number)) => void;
};
export function ReviewQueuePanel(props: Props) {
  const { t } = useTranslation();
  const {
    loading,
    error,
    rows,
    total,
    rangeStart,
    rangeEnd,
    canGoBack,
    canGoForward,
    severity,
    setSeverity,
    status,
    setStatus,
    searchText,
    changeSearch,
    selectedId,
    setSelectedId,
    setOffset,
  } = props;
  return (
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
            <Field label={t("workbench.review.severityFilter", { defaultValue: "Severity" })}>
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

          {loading ? (
            <div className="wb-stack tight" aria-hidden>
              <Skeleton height={78} />
              <Skeleton height={78} />
              <Skeleton height={78} />
            </div>
          ) : error ? (
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
                  aria-pressed={finding.id === selectedId}
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

          {!loading && !error && rows.length > 0 ? (
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
  );
}
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

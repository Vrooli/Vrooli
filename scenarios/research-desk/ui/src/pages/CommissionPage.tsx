import { Alert } from "@vrooli/react-component-library/Alert/1";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { Ring, StatusPill } from "../components/daylight";
import { DerivedWhen, When } from "../components/When";
import { Link, useParams, useSearchParams } from "react-router-dom";

import { CommissionState, useCommission, type GetCommissionResponse } from "../api/activity";
import { QueryView } from "../components/QueryView";
import { StateBadge } from "../components/StateBadge";
import { Unreadable } from "../components/Unreadable";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { toDate } from "../lib/format";
import { displayTitle } from "../lib/titles";
import { useTranslation } from "../i18n";
import { pastAllowance } from "../lib/states";
import { FilesPanel } from "./commission/FilesPanel";
import { ReadingPanel } from "./commission/ReadingPanel";
import { StatusPanel } from "./commission/StatusPanel";
import { SteerPanel } from "./commission/SteerPanel";

/** How much of the allowance is left, as a small pill beside the state. */
function AllowancePill({ commission }: { commission: NonNullable<GetCommissionResponse["commission"]> }) {
  const { t } = useTranslation();
  const end = toDate(commission.allowanceEnd?.value);
  if (!end) return <StatusPill tone="quiet">{commission.allowanceEnd?.evidence?.reason || t(strings.commission.allowanceUnknown)}</StatusPill>;
  const remaining = end.getTime() - Date.now();
  if (remaining <= 0) return <StatusPill tone="peach" dot="wait">{t(strings.commission.allowanceEndedShort)}</StatusPill>;
  const hours = Math.ceil(remaining / 3_600_000);
  return <StatusPill tone="mint" dot="live">{hours < 24 ? t(strings.commission.hoursLeft, { count: hours }) : t(strings.commission.daysLeft, { count: Math.ceil(hours / 24) })}</StatusPill>;
}

/** The one white ring card: time left, last write, start and supervisor. */
function CommissionHero({ data }: { data: GetCommissionResponse }) {
  const { t } = useTranslation();
  const c = data.commission;
  if (!c) return null;
  const start = toDate(c.started?.value);
  const end = toDate(c.allowanceEnd?.value);
  const now = Date.now();
  const totalMs = start && end ? end.getTime() - start.getTime() : null;
  const remainingMs = end ? end.getTime() - now : null;
  const percent = totalMs && remainingMs !== null ? Math.min(100, Math.max(0, (remainingMs / totalMs) * 100)) : 0;
  const hoursLeft = remainingMs !== null ? Math.max(0, Math.ceil(remainingMs / 3_600_000)) : null;
  const totalHours = totalMs ? Math.max(1, Math.round(totalMs / 3_600_000)) : null;
  const over = remainingMs !== null && remainingMs <= 0;
  return (
    <div className="rd-cm-hero">
      <Ring
        size="lg"
        value={percent}
        tone={over ? "amber" : percent <= 20 ? "amber" : "blue"}
        label={hoursLeft === null ? "—" : <>{over ? "0h" : `${hoursLeft}h`}{totalHours ? <small>{over ? "allowance ended" : `of ${totalHours}h`}</small> : null}</>}
      />
      <div className="rd-cm-stats">
        <div className="rd-cm-stat"><span>{t(strings.commission.lastWriteShort)}</span><b>{c.lastWrite?.value ? <DerivedWhen time={c.lastWrite} /> : t(strings.common.unknown)}</b></div>
        <div className="rd-cm-stat"><span>{t(strings.commission.started)}</span><b>{start ? <When date={start} /> : t(strings.common.unknown)}</b></div>
        <div className="rd-cm-stat"><span>{t(strings.commission.allowanceEnd)}</span><b>{end ? <When date={end} /> : t(strings.common.unknown)}</b></div>
        <div className="rd-cm-stat"><span>{t(strings.commission.supervisorShort)}</span><b>{c.launchedBy || t(strings.common.unknown)}</b></div>
      </div>
    </div>
  );
}

const TABS = ["status", "steer", "reading", "files"] as const;
type Tab = (typeof TABS)[number];

/**
 * One commission: its derived state with every source a tap away, the
 * operator's words and controls (none of which type into the chat except the
 * named controls), what its agents read to start, and its files.
 */
export function CommissionPage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const tab: Tab = TABS.includes(params.get("tab") as Tab) ? (params.get("tab") as Tab) : "status";
  const query = useCommission(id);
  const labels: Record<Tab, string> = {
    status: t(strings.commission.tabs.status),
    steer: t(strings.commission.tabs.steer),
    reading: t(strings.commission.tabs.reading),
    files: t(strings.commission.tabs.files),
  };
  return (
    <div data-testid={selectors.pages.commission} className="rd-page">
      <Link to="/commissions" className="rd-back">{t(strings.commission.back)}</Link>
      <QueryView query={query}>
        {(data) => {
          const c = data.commission;
          if (!c) return <Alert tone="danger" title={t(strings.commission.missing)} />;
          const flagged = data.readSets.some((set) => set.flags.length > 0);
          return (
            <>
              <PageHeader
                headingId="commission-heading"
                title={<span title={c.title?.value || c.id}>{displayTitle(c.title?.value, c.id)}</span>}
                eyebrow={
                  <span className="inline-flex max-w-full items-center gap-space-3xs">
                    <span className="rd-id">{c.id}</span>
                    <CopyIconButton value={c.id} aria-label={t(strings.common.copyId)} copiedLabel={t(strings.common.copied)} />
                  </span>
                }
                description={
                  <span className="rd-toolbar">
                    <StateBadge state={c.state} stalled={c.stalled} pastAllowance={pastAllowance(c)} />
                    <AllowancePill commission={c} />
                  </span>
                }
              />
              <CommissionHero data={data} />
              <Unreadable evidence={data.unreadable} />
              {data.recordErrors.map((error) => (
                <Alert key={error} tone="warning" title={t(strings.commission.recordError)} description={error} />
              ))}
              {c.state === CommissionState.PREPARED ? (
                <Alert
                  tone="info"
                  title={t(strings.commission.preparedTitle)}
                  actions={<Link to={`/launch/${encodeURIComponent(c.id)}`} className="rd-more">{t(strings.now.reviewAndLaunch)}</Link>}
                />
              ) : null}
              <Tabs
                mode="controlled"
                variant="segmented"
                ariaLabel={t(strings.commission.tabs.label)}
                items={TABS.map((key) => ({ id: key, label: labels[key], badge: key === "reading" && flagged ? "!" : undefined }))}
                active={tab}
                onChange={(next) => setParams((prev) => {
                  const out = new URLSearchParams(prev);
                  out.set("tab", next);
                  return out;
                }, { replace: true })}
                panels={{
                  status: <StatusPanel data={data} />,
                  steer: <SteerPanel data={data} />,
                  reading: <ReadingPanel data={data} />,
                  files: <FilesPanel data={data} />,
                }}
              />
            </>
          );
        }}
      </QueryView>
    </div>
  );
}

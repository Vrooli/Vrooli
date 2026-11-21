import { Alert } from "@vrooli/react-component-library/Alert/1";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { Link, useParams, useSearchParams } from "react-router-dom";

import { CommissionState, useCommission } from "../api/activity";
import { QueryView } from "../components/QueryView";
import { StateBadge } from "../components/StateBadge";
import { Unreadable } from "../components/Unreadable";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { pastAllowance } from "../lib/states";
import { FilesPanel } from "./commission/FilesPanel";
import { ReadingPanel } from "./commission/ReadingPanel";
import { StatusPanel } from "./commission/StatusPanel";
import { SteerPanel } from "./commission/SteerPanel";

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
                title={c.title?.value || c.id}
                eyebrow={
                  <span className="inline-flex max-w-full items-center gap-space-3xs">
                    <span className="rd-id">{c.id}</span>
                    <CopyIconButton value={c.id} aria-label={t(strings.common.copyId)} copiedLabel={t(strings.common.copied)} />
                  </span>
                }
                description={<StateBadge state={c.state} stalled={c.stalled} pastAllowance={pastAllowance(c)} />}
              />
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

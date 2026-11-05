import { useCallback, useEffect, useMemo, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import {
  DataTable,
  type DataTableColumn,
  type DataTableFilter,
  type DataTableStatus,
} from "@vrooli/react-component-library/DataTable/1";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import type { Capability } from "@vrooli/proto-types/content-desk/v1/capabilities/capabilities_pb";
import type { Campaign } from "@vrooli/proto-types/content-desk/v1/campaigns/campaigns_pb";
import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";

import { artifactsClient } from "../api/artifacts";
import { readBoard, type BoardLaunchTarget } from "../api/board";
import { campaignsClient } from "../api/campaigns";
import { capabilitiesClient } from "../api/capabilities";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { capabilityGaps, draftNextActionKey, isTerminalDraft, type CapabilityGap, type NextActionKey } from "../features/board/boardModel";
import { useTranslation } from "../i18n";

type BoardState = "loading" | "ready" | "error";

interface WorkRow {
  id: string;
  campaignId: string;
  campaignName: string;
  channel: string;
  status: string;
  nextActionKey: NextActionKey;
}

function statusTone(status: string): StatusTone {
  switch (status) {
    case "published":
    case "approved":
      return "success";
    case "reviewed":
      return "info";
    case "blocked":
      return "danger";
    case "abandoned":
      return "neutral";
    default:
      return "warning";
  }
}

export function BoardPage() {
  const { t } = useTranslation();
  const [state, setState] = useState<BoardState>(() => (import.meta.env.MODE === "test" ? "ready" : "loading"));
  const [error, setError] = useState("");
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [capabilities, setCapabilities] = useState<Capability[]>([]);
  const [offerReadStatus, setOfferReadStatus] = useState("unavailable");
  const [offerTargets, setOfferTargets] = useState<BoardLaunchTarget[]>([]);
  const [offerError, setOfferError] = useState("");

  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const [draftResponse, campaignResponse, capabilityResponse] = await Promise.all([
        artifactsClient.listDrafts({}),
        campaignsClient.listCampaigns({}),
        capabilitiesClient.listCapabilities({}),
      ]);
      setDrafts(draftResponse.drafts);
      setCampaigns(campaignResponse.campaigns);
      setCapabilities(capabilityResponse.capabilities);
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.board.error));
      setState("error");
    }
    try {
      const envelope = await readBoard();
      setOfferReadStatus(envelope.signals.offer_read_status ?? "unavailable");
      setOfferTargets(envelope.signals.offer_readiness?.launch_targets ?? []);
      setOfferError("");
    } catch (caught) {
      setOfferReadStatus("unavailable");
      setOfferTargets([]);
      setOfferError(caught instanceof Error ? caught.message : t(strings.board.offerUnavailable));
    }
  }, [t]);

  useEffect(() => {
    if (import.meta.env.MODE !== "test") void refresh();
  }, [refresh]);

  const campaignNames = useMemo(() => {
    const names = new Map<string, string>();
    for (const campaign of campaigns) names.set(campaign.id, campaign.name || campaign.id);
    return names;
  }, [campaigns]);

  const workRows = useMemo<WorkRow[]>(
    () =>
      drafts
        .filter((draft) => !isTerminalDraft(draft.status))
        .map((draft) => ({
          id: draft.id,
          campaignId: draft.campaignId,
          campaignName: campaignNames.get(draft.campaignId) ?? draft.campaignId,
          channel: draft.channel,
          status: draft.status,
          nextActionKey: draftNextActionKey(draft.status),
        })),
    [campaignNames, drafts],
  );

  const gaps = useMemo<CapabilityGap[]>(() => capabilityGaps(capabilities), [capabilities]);
  const activeCampaigns = useMemo(() => campaigns.filter((campaign) => campaign.status === "active").length, [campaigns]);

  const workColumns = useMemo<Array<DataTableColumn<WorkRow>>>(
    () => [
      {
        id: "draft",
        header: t(strings.board.columns.draft),
        accessor: (row) => <span className="font-mono text-sm">{row.id}</span>,
        sortValue: (row) => row.id,
      },
      {
        id: "campaign",
        header: t(strings.board.columns.campaign),
        accessor: (row) => row.campaignName,
        sortValue: (row) => row.campaignName,
      },
      {
        id: "channel",
        header: t(strings.board.columns.channel),
        accessor: (row) => row.channel,
        sortValue: (row) => row.channel,
      },
      {
        id: "status",
        header: t(strings.board.columns.status),
        accessor: (row) => <StatusBadge tone={statusTone(row.status)}>{row.status}</StatusBadge>,
        sortValue: (row) => row.status,
      },
      {
        id: "nextAction",
        header: t(strings.board.columns.nextAction),
        accessor: (row) => t(row.nextActionKey),
        sortValue: (row) => row.nextActionKey,
      },
    ],
    [t],
  );

  const workFilters = useMemo<Array<DataTableFilter<WorkRow>>>(() => {
    const statuses = Array.from(new Set(workRows.map((row) => row.status))).sort();
    return statuses.map((status) => ({
      id: `status:${status}`,
      label: status,
      predicate: (row: WorkRow) => row.status === status,
    }));
  }, [workRows]);

  const gapColumns = useMemo<Array<DataTableColumn<CapabilityGap>>>(
    () => [
      {
        id: "capability",
        header: t(strings.board.columns.capability),
        accessor: (row) => <span className="font-medium">{row.name}</span>,
        sortValue: (row) => row.name,
      },
      {
        id: "readiness",
        header: t(strings.board.columns.readiness),
        accessor: (row) => (
          <span className="flex flex-wrap gap-1">
            {row.dimensions.map((dimension) => (
              <StatusBadge key={dimension.key} tone="warning">
                {`${t(dimension.label)}: ${dimension.value}`}
              </StatusBadge>
            ))}
          </span>
        ),
        sortValue: (row) => row.dimensions.map((dimension) => dimension.key).join(","),
      },
      {
        id: "owner",
        header: t(strings.board.columns.owner),
        accessor: (row) => row.owner,
        sortValue: (row) => row.owner,
      },
      {
        id: "nextCapabilityAction",
        header: t(strings.board.columns.nextCapabilityAction),
        accessor: (row) => row.nextAction,
        sortValue: (row) => row.nextAction,
      },
    ],
    [t],
  );

  const offerColumns = useMemo<Array<DataTableColumn<BoardLaunchTarget>>>(
    () => [
      {
        id: "offerScenario",
        header: t(strings.board.columns.offerScenario),
        accessor: (row) => <span className="font-medium">{row.scenario}</span>,
        sortValue: (row) => row.scenario,
      },
      {
        id: "offerStatus",
        header: t(strings.board.columns.offerStatus),
        accessor: (row) => (
          <StatusBadge tone={row.status === "TRIGGER_MET" || row.status === "RETIRED" ? "success" : "warning"}>
            {row.status}
          </StatusBadge>
        ),
        sortValue: (row) => row.status,
      },
      {
        id: "offerRank",
        header: t(strings.board.columns.offerRank),
        accessor: (row) => row.release_rank,
        sortValue: (row) => row.release_rank,
      },
      {
        id: "offerBlockedBy",
        header: t(strings.board.columns.offerBlockedBy),
        accessor: (row) => row.blocked_by.map((entry) => `${entry.name} (${entry.status})`).join(", "),
        sortValue: (row) => row.blocked_by.length,
      },
      {
        id: "offerNextAction",
        header: t(strings.board.columns.nextAction),
        accessor: (row) => row.next_action,
        sortValue: (row) => row.next_action,
      },
    ],
    [t],
  );

  const workStatus: DataTableStatus = state === "loading" ? "loading" : state === "error" ? "request-error" : workRows.length === 0 ? "empty" : "success";
  const gapStatus: DataTableStatus = state === "loading" ? "loading" : state === "error" ? "request-error" : gaps.length === 0 ? "empty" : "success";
  const offerStatus: DataTableStatus = state === "loading" ? "loading" : offerError ? "request-error" : offerTargets.length === 0 ? "empty" : "success";

  return (
    <section data-testid={selectors.pages.board} aria-labelledby="board-heading" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 id="board-heading" className="text-2xl font-semibold">{t(strings.board.title)}</h2>
          <p className="text-app-muted-foreground">{t(strings.board.subtitle)}</p>
        </div>
        <Button variant="secondary" data-testid={selectors.pages.boardRefresh} onClick={() => void refresh()} disabled={state === "loading"}>
          {t(strings.board.refresh)}
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card><CardHeader><CardTitle>{t(strings.board.summary.currentWork)}</CardTitle></CardHeader><CardContent><p className="text-3xl font-semibold">{workRows.length}</p></CardContent></Card>
        <Card><CardHeader><CardTitle>{t(strings.board.summary.activeCampaigns)}</CardTitle></CardHeader><CardContent><p className="text-3xl font-semibold">{activeCampaigns}</p></CardContent></Card>
        <Card><CardHeader><CardTitle>{t(strings.board.summary.capabilities)}</CardTitle></CardHeader><CardContent><p className="text-3xl font-semibold">{capabilities.length}</p></CardContent></Card>
        <Card><CardHeader><CardTitle>{t(strings.board.summary.gaps)}</CardTitle></CardHeader><CardContent><p className="text-3xl font-semibold">{gaps.length}</p></CardContent></Card>
      </div>

      <Card>
        <CardHeader><CardTitle>{t(strings.board.currentWorkTitle)}</CardTitle></CardHeader>
        <CardContent>
          <DataTable
            rows={workRows}
            columns={workColumns}
            getRowKey={(row) => row.id}
            caption={t(strings.board.currentWorkCaption)}
            filters={workFilters}
            filterLabel={t(strings.board.filterLabel)}
            status={workStatus}
            statusMessage={t(strings.board.loading)}
            errorMessage={error || t(strings.board.error)}
            emptyMessage={t(strings.board.emptyTitle)}
            emptyDetail={t(strings.board.emptyDetail)}
            onRetry={() => void refresh()}
            defaultDensity="compact"
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t(strings.board.capabilityTitle)}</CardTitle></CardHeader>
        <CardContent>
          <DataTable
            rows={gaps}
            columns={gapColumns}
            getRowKey={(row) => row.id}
            caption={t(strings.board.capabilityCaption)}
            status={gapStatus}
            statusMessage={t(strings.board.loading)}
            errorMessage={error || t(strings.board.error)}
            emptyMessage={t(strings.board.emptyTitle)}
            emptyDetail={t(strings.board.emptyDetail)}
            onRetry={() => void refresh()}
            defaultDensity="compact"
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>{t(strings.board.offerTitle)}</CardTitle></CardHeader>
        <CardContent>
          <DataTable
            rows={offerTargets}
            columns={offerColumns}
            getRowKey={(row) => row.node_id || row.scenario}
            caption={t(strings.board.offerCaption)}
            status={offerStatus}
            statusMessage={t(strings.board.loading)}
            errorMessage={offerError || t(strings.board.offerUnavailable)}
            emptyMessage={offerReadStatus === "read" ? t(strings.board.offerEmpty) : t(strings.board.offerUnavailable)}
            emptyDetail={t(strings.board.offerCaption)}
            onRetry={() => void refresh()}
            defaultDensity="compact"
          />
        </CardContent>
      </Card>
    </section>
  );
}

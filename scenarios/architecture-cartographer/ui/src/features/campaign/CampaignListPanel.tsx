import * as React from "react";
import { Link, useNavigate } from "react-router-dom";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { useListCampaigns } from "./controllers/useCampaignController";
import { CreateCampaignForm } from "./CreateCampaignForm";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import {
  CampaignLifecycle,
  type Campaign,
} from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

export interface CampaignListPanelProps {
  scenario: string;
  /** When provided, the matching row renders as selected. */
  selectedId?: string;
}

function lifecycleLabelKey(status: CampaignLifecycle) {
  return status === CampaignLifecycle.CLOSED
    ? strings.campaign.lifecycle.closed
    : strings.campaign.lifecycle.open;
}

export function CampaignListPanel({ scenario, selectedId }: CampaignListPanelProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const list = useListCampaigns({ scenario });
  const [creating, setCreating] = React.useState(false);

  const columns: ReadonlyArray<DataTableColumn<Campaign>> = [
    {
      key: "id",
      header: t(strings.pages.campaign.columns.id),
      cell: (row) => {
        const isSelected = selectedId !== undefined && row.id === selectedId;
        return (
          <Link
            to={`/targets/${encodeScenarioPath(scenario)}/campaign/${encodeURIComponent(row.id)}`}
            data-testid={selectors.features.campaign.list.openButton({ id: row.id })}
            className={`block font-mono text-xs ${isSelected ? "text-app-primary" : "text-app-foreground hover:underline"}`}
          >
            {row.id.slice(0, 8)}
          </Link>
        );
      },
    },
    {
      key: "name",
      header: t(strings.pages.campaign.columns.name),
      cell: (row) => <span className="text-sm">{row.name || t(strings.pages.campaign.unnamed)}</span>,
    },
    {
      key: "status",
      header: t(strings.pages.campaign.columns.status),
      cell: (row) => (
        <Badge variant={row.status === CampaignLifecycle.CLOSED ? "default" : "info"}>
          {t(lifecycleLabelKey(row.status))}
        </Badge>
      ),
    },
    {
      key: "created",
      header: t(strings.pages.campaign.columns.created),
      cell: (row) => (
        <span className="text-xs text-app-muted-foreground">
          {row.createdAt ? formatDate(timestampDate(row.createdAt), { dateStyle: "medium", timeStyle: "short" }) : "—"}
        </span>
      ),
    },
  ];

  if (list.isPending) {
    return (
      <div data-testid={selectors.features.campaign.list.loading}>
        <LoadingState label={t(strings.pages.campaign.loading)} />
      </div>
    );
  }

  if (list.isError) {
    return (
      <div data-testid={selectors.features.campaign.list.error}>
        <ErrorState
          title={t(strings.pages.campaign.errorTitle)}
          message={list.error instanceof Error ? list.error.message : String(list.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void list.refetch();
          }}
        />
      </div>
    );
  }

  const campaigns = list.data.campaigns;

  return (
    <section
      data-testid={selectors.features.campaign.list.root}
      aria-label={t(strings.pages.campaign.title)}
      className="flex flex-col gap-3"
    >
      <header className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.campaign.list.newButton}
          onClick={() => setCreating((v) => !v)}
        >
          {t(strings.pages.campaign.newButton)}
        </Button>
      </header>

      {creating ? (
        <CreateCampaignForm
          scenario={scenario}
          onCreated={(id) => {
            setCreating(false);
            navigate(`/targets/${encodeScenarioPath(scenario)}/campaign/${encodeURIComponent(id)}`);
          }}
          onCancel={() => setCreating(false)}
        />
      ) : null}

      {campaigns.length === 0 ? (
        <div data-testid={selectors.features.campaign.list.empty}>
          <EmptyState
            title={t(strings.pages.campaign.listEmptyTitle)}
            description={t(strings.pages.campaign.listEmptyDescription)}
          />
        </div>
      ) : (
        <DataTable<Campaign>
          rows={campaigns}
          getRowId={(row) => row.id}
          columns={columns}
          emptyMessage={t(strings.pages.campaign.listEmptyTitle)}
          caption={t(strings.pages.campaign.title)}
        />
      )}
    </section>
  );
}

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { SplitPane } from "../../components/SplitPane";
import { CampaignDetailPanel } from "./CampaignDetailPanel";
import { CampaignListPanel } from "./CampaignListPanel";

export interface CampaignWorkbenchProps {
  scenario: string;
  /** When provided, the detail pane focuses on this campaign. */
  campaignId?: string;
}

/**
 * Two-pane workbench: the scenario's campaigns on the primary side, the
 * selected campaign's items + lifecycle on the secondary side. Mirrors
 * the (now detection-only) conflict workbench shape so the two surfaces feel
 * like siblings: conflicts shows what's wrong *now*, campaign tracks the
 * improvement effort that drives it to zero over time.
 */
export function CampaignWorkbench({ scenario, campaignId }: CampaignWorkbenchProps) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.features.campaign.workbench.root} className="flex flex-col gap-3">
      <SplitPane
        handleLabel={t(strings.shared.splitPane.resizeHandle)}
        initialPercent={40}
        primary={<CampaignListPanel scenario={scenario} selectedId={campaignId} />}
        secondary={
          campaignId ? (
            <CampaignDetailPanel scenario={scenario} campaignId={campaignId} />
          ) : (
            <div data-testid={selectors.features.campaign.workbench.emptyDetail}>
              <EmptyState title={t(strings.pages.campaign.selectPrompt)} />
            </div>
          )
        }
      />
    </div>
  );
}

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { SplitPane } from "../../components/SplitPane";
import { MigrationDetailPanel } from "./MigrationDetailPanel";
import { MigrationListPanel } from "./MigrationListPanel";

export interface MigrationWorkbenchProps {
  scenario: string;
  /** When provided, the detail pane focuses on this migration. */
  migrationId?: string;
}

/**
 * Two-pane workbench: the scenario's migrations on the primary side, the
 * selected migration's findings + lifecycle on the secondary side. Mirrors
 * the (now detection-only) conflict workbench shape so the two surfaces feel
 * like siblings: conflicts shows what's wrong *now*, migration tracks the
 * refactor that drives it to zero over time.
 */
export function MigrationWorkbench({ scenario, migrationId }: MigrationWorkbenchProps) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.features.migration.workbench.root} className="flex flex-col gap-3">
      <SplitPane
        handleLabel={t(strings.shared.splitPane.resizeHandle)}
        initialPercent={40}
        primary={<MigrationListPanel scenario={scenario} selectedId={migrationId} />}
        secondary={
          migrationId ? (
            <MigrationDetailPanel scenario={scenario} migrationId={migrationId} />
          ) : (
            <div data-testid={selectors.features.migration.workbench.emptyDetail}>
              <EmptyState title={t(strings.pages.migration.selectPrompt)} />
            </div>
          )
        }
      />
    </div>
  );
}

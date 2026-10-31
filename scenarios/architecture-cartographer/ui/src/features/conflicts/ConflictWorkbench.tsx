import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { SplitPane } from "../../components/SplitPane";
import { ConflictDetailPanel } from "./ConflictDetailPanel";
import { ConflictListPanel } from "./ConflictListPanel";

export interface ConflictWorkbenchProps {
  scenario: string;
  /** When provided, the detail pane focuses on this conflict. */
  conflictId?: string;
}

/**
 * Two-pane workbench: list on the primary side, detail on the secondary side.
 * On mobile the SplitPane stacks the two; consumers expecting a stepwise-tab
 * mobile experience can wrap with the shared `Tabs` primitive in a later
 * refinement.
 */
export function ConflictWorkbench({ scenario, conflictId }: ConflictWorkbenchProps) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.features.conflicts.workbench.root}
      className="flex flex-col gap-3"
    >
      <SplitPane
        handleLabel={t(strings.shared.splitPane.resizeHandle)}
        initialPercent={45}
        primary={<ConflictListPanel scenario={scenario} selectedId={conflictId} />}
        secondary={
          conflictId ? (
            <ConflictDetailPanel scenario={scenario} conflictId={conflictId} />
          ) : (
            <div data-testid={selectors.features.conflicts.workbench.emptyDetail}>
              <EmptyState title={t(strings.pages.conflicts.selectConflictPrompt)} />
            </div>
          )
        }
      />
    </div>
  );
}

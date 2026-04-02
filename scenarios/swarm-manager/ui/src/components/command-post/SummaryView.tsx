/**
 * SummaryView — The landing/triage view inside the Command Post overlay.
 *
 * Consumes stores directly, computes action groups via groupActionItems(),
 * and renders ActionGroupCards, a prioritized ActionFeedItem list,
 * snoozed section, and recent activity section.
 */

import { useMemo, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBacklogStore } from "../../stores/backlog-store";
import { useExecutionStore } from "../../stores/execution-store";
import { useCaptureStore } from "../../stores/capture-store";
import { useSnoozeStore, useSnoozedKeys } from "../../stores/snooze-store";
import { groupActionItems, type ActionGroup, type ActionGroupId } from "../../lib/command-post-utils";
import { backlogService } from "../../services";
import type { FeedbackItem, MaturityItem } from "../../lib/feed";
import type { DetailSelection } from "../../stores/detail-selection-store";
import type { RunBacklogTarget } from "../backlog/run-backlog-modal";
import { RunBacklogModal } from "../backlog/run-backlog-modal";
import { ActionGroupCard } from "./ActionGroupCard";
import { ActionFeedItem } from "./ActionFeedItem";
import { SnoozedSection } from "./SnoozedSection";
import { RecentSection } from "./RecentSection";
import { EmptyState } from "./EmptyState";

interface SummaryViewProps {
  onEnterDecisionStream: () => void;
  onNavigateToDetail: (selection: DetailSelection) => void;
  onSwitchLens: (lens: string) => void;
  onClose: () => void;
}

export function SummaryView({
  onEnterDecisionStream,
  onNavigateToDetail,
  onSwitchLens,
  onClose,
}: SummaryViewProps) {
  const backlogItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const snoozeEntries = useSnoozeStore((s) => s.entries);
  const snooze = useSnoozeStore((s) => s.snooze);
  const unsnooze = useSnoozeStore((s) => s.unsnooze);
  const snoozedKeys = useSnoozedKeys();

  const [runModalTargets, setRunModalTargets] = useState<RunBacklogTarget[] | undefined>();

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  const feedbackMap = useMemo(() => {
    const map = new Map<string, FeedbackItem>();
    for (const item of summaryQuery.data?.feedback?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        pendingDecisions: item.pending_decisions ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.feedback]);

  const maturityMap = useMemo(() => {
    const map = new Map<string, MaturityItem>();
    for (const item of summaryQuery.data?.maturity?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        ready: item.ready ?? false,
        pendingItems: item.pending_items ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  const groups = useMemo(
    () => groupActionItems(backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys),
    [backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys],
  );

  const allItems = useMemo(() => groups.flatMap((g) => g.items), [groups]);
  const totalCount = useMemo(() => groups.reduce((sum, g) => sum + g.count, 0), [groups]);

  const snoozedEntries = useMemo(
    () => Array.from(snoozeEntries.values()),
    [snoozeEntries],
  );

  const handleSnooze = useCallback(
    (key: string, expiresAt: number) => {
      snooze(key, expiresAt);
    },
    [snooze],
  );

  const handleBulkAction = useCallback(
    (group: ActionGroup) => {
      switch (group.id as ActionGroupId) {
        case "ready-to-run": {
          const targets: RunBacklogTarget[] = group.items
            .filter((i) => i.kind && i.name)
            .map((i) => ({ kind: i.kind as RunBacklogTarget["kind"], name: i.name ?? "", title: i.title }));
          if (targets.length > 0) {
            setRunModalTargets(targets);
          }
          break;
        }
        case "pending-decisions":
          onEnterDecisionStream();
          break;
        case "needs-review":
        case "failed":
        case "needs-classification": {
          const first = group.items[0];
          if (first) navigateToItem(first);
          break;
        }
      }
    },
    [onEnterDecisionStream], // eslint-disable-line react-hooks/exhaustive-deps
  );

  const navigateToItem = useCallback(
    (item: { type: string; kind?: string; name?: string; executionId?: string }) => {
      if (item.type === "backlog" && item.kind && item.name) {
        onNavigateToDetail({ entityType: "backlog", kind: item.kind, name: item.name });
      } else if (item.type === "execution" && item.executionId) {
        onNavigateToDetail({ entityType: "execution", identifier: item.executionId });
      }
      onClose();
    },
    [onNavigateToDetail, onClose],
  );

  const handleRunItem = useCallback(
    (kind: string, name: string) => {
      setRunModalTargets([{ kind: kind as RunBacklogTarget["kind"], name }]);
    },
    [],
  );

  if (totalCount === 0) {
    return <EmptyState onSwitchLens={onSwitchLens} />;
  }

  return (
    <div className="space-y-4" data-testid="command-post-summary">
      {/* Action group cards */}
      <div className="flex flex-wrap gap-3">
        {groups.map((group) => (
          <ActionGroupCard
            key={group.id}
            group={group}
            onBulkAction={() => handleBulkAction(group)}
          />
        ))}
      </div>

      {/* Prioritized feed */}
      <div className="space-y-2">
        <h3 className="text-sm font-medium text-slate-400">Needs Attention</h3>
        {allItems.map((item) => (
          <ActionFeedItem
            key={item.key}
            item={item}
            onNavigate={() => navigateToItem(item)}
            onSnooze={handleSnooze}
            onRun={handleRunItem}
            onEnterDecisionStream={onEnterDecisionStream}
          />
        ))}
      </div>

      {/* Snoozed section */}
      <SnoozedSection
        entries={snoozedEntries}
        onUnsnooze={unsnooze}
      />

      {/* Recent activity */}
      <RecentSection />

      {/* Run modal */}
      <RunBacklogModal
        isOpen={!!runModalTargets}
        onClose={() => setRunModalTargets(undefined)}
        targets={runModalTargets}
        onSuccess={() => setRunModalTargets(undefined)}
      />
    </div>
  );
}

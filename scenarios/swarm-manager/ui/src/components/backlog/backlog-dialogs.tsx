/**
 * BacklogDialogs
 *
 * Container component rendering all dialogs/modals for the BacklogDetailsPage.
 * Reads dialog open/close state from backlog-detail-ui-store.
 */

import { BacklogFormDialog } from "./backlog-form-dialog";
import { AcceptanceGlobDialog } from "./acceptance-glob-dialog";
import { BacklogAgentDialog } from "./backlog-agent-dialog";
import { RunBacklogModal } from "./run-backlog-modal";
import { RequirementFormDialog } from "./requirement-form-dialog";
import { ModuleFormDialog } from "./module-form-dialog";
import { TargetFormDialog } from "./target-form-dialog";
import { ClarificationPanel } from "./clarification-panel";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Drawer } from "../ui/drawer";
import { FollowUpDialog } from "../execution/follow-up-dialog";
import { ActivityTimeline } from "../detail/ActivityTimeline";
import { findRequirementGroup } from "../../lib/archive-utils";
import { selectors } from "../../consts/selectors";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";
import { useBacklogDetailUIStore } from "../../stores";
import type { useBacklogDetailData } from "../../hooks/useBacklogDetailData";
import type { useBacklogHandlers } from "../../hooks/useBacklogHandlers";
import type { useActivityTimeline } from "../../hooks/useActivityTimeline";
import type { BacklogFile, BacklogKind } from "../../types";
import type { BacklogItem } from "../../types/domain";
import type { ReadinessIndicatorData } from "../../lib/maturity";

export interface BacklogDialogsProps {
  data: ReturnType<typeof useBacklogDetailData>;
  handlers: ReturnType<typeof useBacklogHandlers>;
  files: BacklogFile[] | undefined;
  readinessData: ReadinessIndicatorData | null;
  agentDialogTargetIds: Set<string>;
  agentDialogRequirementIds: Set<string>;
  timeline: ReturnType<typeof useActivityTimeline>;
  agentManagerUiUrl: string | null;
  upsertItem: (item: BacklogItem) => void;
  closeDetail: () => void;
  stopRun: (runId: string) => Promise<void>;
}

export function BacklogDialogs({
  data,
  handlers,
  files,
  readinessData,
  agentDialogTargetIds,
  agentDialogRequirementIds,
  timeline,
  agentManagerUiUrl,
  upsertItem,
  closeDetail,
  stopRun,
}: BacklogDialogsProps) {
  const { backlogKind, name, item, agentRunIsActive, latestAgentActivity } = useBacklogDetail();
  const ui = useBacklogDetailUIStore();

  return (
    <>
      {item && (
        <BacklogFormDialog
          isOpen={ui.showEdit}
          mode="edit"
          initialValues={{
            name: item.name,
            title: item.title,
            description: item.description,
            status: item.status,
            priority: item.priority,
            tags: item.tags,
            kind: item.kind,
          }}
          isSubmitting={data.isUpdating}
          submitError={data.updateError}
          onClose={() => { ui.closeEdit(); data.resetUpdateMutation(); }}
          onSubmit={(values) =>
            handlers.handleUpdateItem({
              title: values.title,
              description: values.description,
              status: values.status,
              priority: values.priority,
              tags: values.tags,
            })
          }
        />
      )}

      {item && (
        <AcceptanceGlobDialog
          isOpen={ui.showGlobDialog}
          onClose={() => { ui.closeGlob(); data.resetGlobMutation(); }}
          initialAllow={item.acceptanceAllow ?? []}
          initialDeny={item.acceptanceDeny ?? []}
          onSave={handlers.handleAcceptanceGlobSave}
          isSubmitting={data.isUpdatingGlob}
        />
      )}

      <ConfirmDialog
        isOpen={ui.showDelete}
        onClose={() => { ui.closeDelete(); data.resetDeleteMutation(); }}
        onConfirm={handlers.handleDeleteConfirm}
        title="Delete Backlog Item"
        description={`Are you sure you want to delete "${item?.title || name}"? This will remove the backlog folder permanently.`}
        confirmationText={item?.name}
        confirmLabel="Delete Item"
        isLoading={data.isDeleting}
        testIds={{
          dialog: selectors.backlogDetails.deleteDialog,
          confirmButton: selectors.backlogDetails.deleteConfirmButton,
          cancelButton: selectors.backlogDetails.deleteCancelButton,
        }}
      />

      <RunBacklogModal
        isOpen={ui.showRunModal}
        onClose={ui.closeRunModal}
        target={backlogKind && name ? { kind: backlogKind, name, title: item?.title } : undefined}
        readinessData={readinessData}
        onSuccess={(result) => {
          if (result.item) upsertItem(result.item);
          data.invalidateItem();
          ui.closeRunModal();
        }}
      />

      <BacklogAgentDialog
        isOpen={ui.showAgentDialog}
        isSubmitting={data.isRunningAgent}
        backlogKind={backlogKind as BacklogKind}
        backlogTitle={item?.title ?? name ?? ""}
        itemStatus={item?.status}
        errorMessage={data.agentError}
        files={files}
        archiveTargets={data.archiveTargets}
        initialSelectedTargetIds={agentDialogTargetIds}
        initialSelectedRequirementIds={agentDialogRequirementIds}
        onClose={() => { ui.closeAgent(); data.resetAgentMutation(); }}
        onSubmit={handlers.handleAgentSubmit}
      />

      <RequirementFormDialog
        isOpen={ui.reqDialog.isOpen}
        mode={ui.reqDialog.editing?.req ? "edit" : "create"}
        initialValues={ui.reqDialog.editing?.req}
        isSubmitting={data.isUpdatingReqs}
        submitError={data.updateReqsError}
        onClose={() => { ui.closeReqDialog(); data.resetUpdateReqsMutation(); }}
        onSubmit={handlers.handleReqDialogSubmit}
      />

      <ModuleFormDialog
        isOpen={ui.moduleDialog.isOpen}
        mode={ui.moduleDialog.mode}
        initialValues={
          ui.moduleDialog.editing && data.archiveTargets
            ? (() => {
                const g = findRequirementGroup(data.archiveTargets.requirements, ui.moduleDialog.editing);
                return g ? { id: g.id, title: g.name, description: "" } : undefined;
              })()
            : undefined
        }
        isSubmitting={data.isCreatingModule || data.isUpdatingModuleMeta}
        submitError={data.createModuleError ?? data.updateModuleMetaError}
        onClose={() => { ui.closeModuleDialog(); data.resetCreateModuleMutation(); data.resetUpdateModuleMetaMutation(); }}
        onSubmit={handlers.handleModuleDialogSubmit}
      />

      <TargetFormDialog
        isOpen={ui.targetDialog.isOpen}
        mode={ui.targetDialog.mode}
        initialValues={ui.targetDialog.editing ?? undefined}
        isSubmitting={data.isCreatingTarget || data.isUpdatingTarget}
        submitError={data.createTargetError ?? data.updateTargetError}
        onClose={() => { ui.closeTargetDialog(); data.resetCreateTargetMutation(); data.resetUpdateTargetMutation(); }}
        onSubmit={handlers.handleTargetDialogSubmit}
      />

      {ui.followUpTarget && (
        <FollowUpDialog
          isOpen={Boolean(ui.followUpTarget)}
          onClose={() => ui.setFollowUpTarget(null)}
          execution={ui.followUpTarget}
          onSuccess={() => {
            ui.setFollowUpTarget(null);
            data.invalidateItem();
          }}
        />
      )}

      <Drawer
        isOpen={ui.isTimelineOpen}
        onClose={ui.closeTimeline}
        title="Activity Timeline"
        description="Executions and agent activities for this backlog item"
        testId={selectors.backlogDetails.activityTimeline}
      >
        <ActivityTimeline
          entries={timeline.entries}
          isLoading={timeline.isLoading}
          error={timeline.error}
          onViewExecution={() => { ui.closeTimeline(); closeDetail(); }}
          onStopRun={(runId) => void stopRun(runId)}
          onFollowUp={(exec) => { ui.closeTimeline(); ui.setFollowUpTarget(exec); }}
          latestAgentActivity={latestAgentActivity ?? undefined}
          agentRunIsActive={agentRunIsActive}
          agentManagerUiUrl={agentManagerUiUrl ?? undefined}
        />
      </Drawer>

      <ClarificationPanel
        onAction={(action) => {
          if (action === "invalidate_round" || action === "remove_decision" || action === "update_decision") {
            data.refetchItem();
          }
        }}
      />
    </>
  );
}

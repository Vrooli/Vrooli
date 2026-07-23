/**
 * BacklogDialogs
 *
 * Container component rendering all dialogs/modals for the BacklogDetailsPage.
 * Reads dialog open/close state from backlog-detail-ui-store.
 */

import { BacklogFormDialog } from "./backlog-form-dialog";
import { AcceptanceGlobDialog } from "./acceptance-glob-dialog";
import { RunSheet } from "./run-sheet";
import { RequirementFormDialog } from "./requirement-form-dialog";
import { ModuleFormDialog } from "./module-form-dialog";
import { TargetFormDialog } from "./target-form-dialog";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { FollowUpSheet } from "../review/follow-up-sheet";
import { findRequirementGroup } from "../../lib/archive-utils";
import { selectors } from "../../consts/selectors";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";
import { useRuntimeConfig } from "../../hooks/useRuntimeConfig";
import { useBacklogDetailUIStore } from "../../stores";
import type { useBacklogDetailData } from "../../hooks/useBacklogDetailData";
import type { useBacklogHandlers } from "../../hooks/useBacklogHandlers";
import type { BacklogItem } from "../../types/domain";

export interface BacklogDialogsProps {
  data: ReturnType<typeof useBacklogDetailData>;
  handlers: ReturnType<typeof useBacklogHandlers>;
  upsertItem: (item: BacklogItem) => void;
}

export function BacklogDialogs({
  data,
  handlers,
  upsertItem,
}: BacklogDialogsProps) {
  const { backlogKind, name, item } = useBacklogDetail();
  const ui = useBacklogDetailUIStore();
  const { getDeleteConfirmLevel } = useRuntimeConfig();
  const deleteLevel = getDeleteConfirmLevel("backlog");

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

      {deleteLevel !== "none" && (
        <ConfirmDialog
          isOpen={ui.showDelete}
          onClose={() => { ui.closeDelete(); data.resetDeleteMutation(); }}
          onConfirm={handlers.handleDeleteConfirm}
          title="Delete Backlog Item"
          description={`Are you sure you want to delete "${item?.title || name}"? This will remove the backlog folder permanently.`}
          confirmationText={deleteLevel === "strong" ? item?.name : undefined}
          confirmLabel="Delete Item"
          isLoading={data.isDeleting}
          testIds={{
            dialog: selectors.backlogDetails.deleteDialog,
            confirmButton: selectors.backlogDetails.deleteConfirmButton,
            cancelButton: selectors.backlogDetails.deleteCancelButton,
            copyButton: selectors.backlogDetails.deleteCopyButton,
          }}
        />
      )}

      <RunSheet
        isOpen={ui.showRunModal}
        onClose={ui.closeRunModal}
        target={backlogKind && name ? { kind: backlogKind, name, title: item?.title } : undefined}
        onSuccess={(result) => {
          if (result.item) upsertItem(result.item);
          data.invalidateItem();
          ui.closeRunModal();
        }}
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
        <FollowUpSheet
          isOpen={Boolean(ui.followUpTarget)}
          onClose={() => ui.setFollowUpTarget(null)}
          execution={ui.followUpTarget}
          reviewRounds={data.reviewRounds}
          initialContext={ui.followUpContext}
          onSuccess={() => {
            ui.setFollowUpTarget(null);
            data.invalidateItem();
          }}
        />
      )}

    </>
  );
}

/**
 * BacklogDialogs
 *
 * Container component rendering all dialogs/modals for the BacklogDetailsPage.
 * Reads dialog open/close state from backlog-detail-ui-store.
 */

import { BacklogFormDialog } from "./backlog-form-dialog";
import { useState } from "react";
import { AcceptanceGlobDialog } from "./acceptance-glob-dialog";
import { BacklogAgentDialog } from "./backlog-agent-dialog";
import { RunBacklogModal } from "./run-backlog-modal";
import { RequirementFormDialog } from "./requirement-form-dialog";
import { ModuleFormDialog } from "./module-form-dialog";
import { TargetFormDialog } from "./target-form-dialog";
import { ClarificationPanel } from "./clarification-panel";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { FollowUpSheet } from "../review/follow-up-sheet";
import { findRequirementGroup } from "../../lib/archive-utils";
import { selectors } from "../../consts/selectors";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";
import { useRuntimeConfig } from "../../hooks/useRuntimeConfig";
import { useBacklogDetailUIStore } from "../../stores";
import { defaultApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type { useBacklogDetailData } from "../../hooks/useBacklogDetailData";
import type { useBacklogHandlers } from "../../hooks/useBacklogHandlers";
import type { BacklogFile, ItemBlockingInfo } from "../../types";
import type { BacklogItem } from "../../types/domain";
import type { ReadinessIndicatorData } from "../../lib/maturity";

const resetScopeOptions: Array<[string, string]> = [["workshop", "Workshop rounds and conclusion"], ["clarifications", "Clarifications"], ["review", "Review rounds"], ["handoff_executions", "Handoff data and executions"], ["plan_unbind", "Plan binding"]];

export interface BacklogDialogsProps {
  data: ReturnType<typeof useBacklogDetailData>;
  handlers: ReturnType<typeof useBacklogHandlers>;
  files: BacklogFile[] | undefined;
  readinessData: ReadinessIndicatorData | null;
  agentDialogTargetIds: Set<string>;
  agentDialogRequirementIds: Set<string>;
  upsertItem: (item: BacklogItem) => void;
  blockingInfo?: ItemBlockingInfo | null;
}

export function BacklogDialogs({
  data,
  handlers,
  files,
  readinessData,
  agentDialogTargetIds,
  agentDialogRequirementIds,
  upsertItem,
  blockingInfo,
}: BacklogDialogsProps) {
  const { backlogKind, name, item, agentRunIsActive, agentRunningLabel } = useBacklogDetail();
  const ui = useBacklogDetailUIStore();
  const { getDeleteConfirmLevel } = useRuntimeConfig();
  const deleteLevel = getDeleteConfirmLevel("backlog");
	const [recreateOpen, setRecreateOpen] = useState(false);
	const [resetOpen, setResetOpen] = useState(false);
	const [scope, setScope] = useState<string[]>(["workshop"]);
	const [lifecyclePending, setLifecyclePending] = useState(false);
	const [lifecycleError, setLifecycleError] = useState<string | undefined>();
	const toggleScope = (value: string) => setScope((current) => current.includes(value) ? current.filter((entry) => entry !== value) : [...current, value]);
	const recreate = async () => {
		if (!backlogKind || !name) return;
		setLifecyclePending(true); setLifecycleError(undefined);
		try { await defaultApiClient.post(API_ENDPOINTS.backlogRecreate(backlogKind, name), {}); setRecreateOpen(false); data.invalidateItem(); }
		catch (error) { setLifecycleError(error instanceof Error ? error.message : "Unable to recreate item."); }
		finally { setLifecyclePending(false); }
	};
	const resetArtifacts = async () => {
		if (!backlogKind || !name || scope.length === 0) return;
		setLifecyclePending(true); setLifecycleError(undefined);
		try { await defaultApiClient.post(API_ENDPOINTS.backlogResetArtifacts(backlogKind, name), { scope }); setResetOpen(false); data.invalidateItem(); }
		catch (error) { setLifecycleError(error instanceof Error ? error.message : "Unable to reset artifacts."); }
		finally { setLifecyclePending(false); }
	};

  return (
    <>
		{item && <section className="mt-6 rounded-lg border border-rose-500/30 bg-rose-500/[0.04] p-4" data-testid="backlog-lifecycle-controls">
			<h2 className="text-sm font-semibold text-rose-200">Lifecycle controls</h2>
			<p className="mt-1 text-xs text-slate-400">{agentRunIsActive ? `${agentRunningLabel} Lifecycle changes are unavailable until it finishes.` : "These actions preserve the canonical spec or archived history, but remove derived work only after confirmation."}</p>
			<div className="mt-3 flex flex-wrap gap-2"><button type="button" className="rounded border border-rose-400/40 px-3 py-1.5 text-sm text-rose-200 disabled:opacity-50" disabled={lifecyclePending || agentRunIsActive} onClick={() => setRecreateOpen(true)}>Recreate item</button><button type="button" className="rounded border border-amber-400/40 px-3 py-1.5 text-sm text-amber-200 disabled:opacity-50" disabled={lifecyclePending || agentRunIsActive} onClick={() => setResetOpen(true)}>Reset artifacts</button></div>
		</section>}
		<ConfirmDialog isOpen={recreateOpen} onClose={() => setRecreateOpen(false)} onConfirm={() => void recreate()} title="Recreate backlog item" description={`Archives "${item?.title || name}" and creates a fresh backlog clone. Metadata, membership, dependencies, and lineage are retained; derived work starts fresh.`} confirmationText={item?.name} confirmLabel="Recreate item" isLoading={lifecyclePending} errorMessage={lifecycleError} />
		<ConfirmDialog isOpen={resetOpen} onClose={() => setResetOpen(false)} onConfirm={() => void resetArtifacts()} title="Reset derived artifacts" description="Deletes only the selected derived artifacts. The canonical item specification is kept." confirmLabel="Reset selected artifacts" isLoading={lifecyclePending} errorMessage={lifecycleError} sidePanel={<div className="space-y-2 p-4"><p className="text-sm font-medium text-slate-100">Choose what to remove</p>{resetScopeOptions.map(([value, label]) => <label key={value} className="flex items-center gap-2 text-sm text-slate-300"><input type="checkbox" checked={scope.includes(value)} onChange={() => toggleScope(value)} />{label}</label>)}</div>} />
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

      <ConfirmDialog
        isOpen={ui.showWorkshopReset}
        onClose={() => { ui.closeWorkshopReset(); data.resetWorkshopResetMutation(); }}
        onConfirm={handlers.handleWorkshopResetConfirm}
        title="Reset Workshop"
        description={`This will delete all workshop rounds, clarifications, attachments, and the ${data.deliverableLabel?.toLowerCase() ?? "deliverable"} for "${item?.title || name}". The item spec will be preserved.`}
        confirmationText={item?.name}
        confirmLabel="Reset Workshop"
        isLoading={data.isResettingWorkshop}
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
        backlogKind={backlogKind}
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
        <FollowUpSheet
          isOpen={Boolean(ui.followUpTarget)}
          onClose={() => ui.setFollowUpTarget(null)}
          execution={ui.followUpTarget}
          reviewRounds={data.reviewRounds}
          onSuccess={() => {
            ui.setFollowUpTarget(null);
            data.invalidateItem();
          }}
        />
      )}

      <ClarificationPanel
        onAction={(action) => {
          if (action === "invalidate_round" || action === "remove_decision" || action === "update_decision") {
            data.refetchItem();
          }
        }}
      />

      <ConfirmDialog
        isOpen={ui.workshopBlockingConfirm.show}
        onClose={ui.closeWorkshopBlockingConfirm}
        onConfirm={handlers.handleWorkshopBlockingOverride}
        title="Dependencies Not Ready"
        description={
          blockingInfo?.blockingDepKeys.length
            ? `This item is blocked by incomplete dependencies: ${blockingInfo.blockingDepKeys.join(", ")}. Do you want to proceed anyway?`
            : "This item has incomplete dependencies. Do you want to proceed anyway?"
        }
        confirmLabel="Override and Proceed"
      />
    </>
  );
}

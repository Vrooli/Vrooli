import { useRef, useState, type HTMLAttributes, type RefObject } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { X } from "lucide-react";
import type { Workspace } from "../../api/workspaces";
import type { WorkspacePage } from "../../api/workspaces";
import { loadComponentBundle, useCatalog, wbKeys } from "../data";
import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../design-types";
import {
  StudioInspector,
  type AiScope,
  type InspectorTab,
  type StudioClaim,
} from "../studio/StudioInspector";
import { asCatalogReplacementNode, asReplacementNode } from "../studio/nodeReplacement";
import { useFrameAudit } from "../studio/useFrameAudit";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import { proposeWorkspacePatch } from "../studio/workspaceAssist";
import { setProp, updateNode, type TemplateNode, type TemplateNodeType } from "../templates/spec";
import type { Coverage } from "../studio/rcl";
import type { StateAudit } from "../studio/checks";

interface WorkspaceStudioInspectorPaneProps {
  studio: ReturnType<typeof useWorkspaceStudio>;
  workspace: Workspace;
  page: WorkspacePage;
  design: DesignDocument;
  frame: DesignFrame;
  frames: DesignFrame[];
  selected: TemplateNode | undefined;
  commitFrame: (
    mutate: (spec: TemplateNode) => TemplateNode,
    label: string,
    mergeKey?: string,
  ) => void;
  coverage: Coverage;
  claims: StudioClaim[];
  audits: ReturnType<typeof useFrameAudit>;
  stateGap: StateAudit;
  rightTab: InspectorTab;
  onRightTab: (tab: InspectorTab) => void;
  aiScope: AiScope;
  onAiScope: (scope: AiScope) => void;
  selectFrame: (id: string) => void;
  flash: (message: string) => void;
  rightSheetRef: RefObject<HTMLElement>;
  inertWhenReadOnly: HTMLAttributes<HTMLElement>;
  onCloseSheet: () => void;
}

export function WorkspaceStudioInspectorPane({
  studio,
  workspace,
  page,
  design,
  frame,
  frames,
  selected,
  commitFrame,
  coverage,
  claims,
  audits,
  stateGap,
  rightTab,
  onRightTab,
  aiScope,
  onAiScope,
  selectFrame,
  flash,
  rightSheetRef,
  inertWhenReadOnly,
  onCloseSheet,
}: WorkspaceStudioInspectorPaneProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const catalog = useCatalog();
  const [pendingCatalogAssetId, setPendingCatalogAssetId] = useState("");
  const selectedLocation = useRef<{ frameId: string; nodeId: string } | null>(null);
  const catalogReplacementRequest = useRef(0);
  selectedLocation.current = selected ? { frameId: frame.id, nodeId: selected.id } : null;
  return (
    <aside
      ref={rightSheetRef}
      {...inertWhenReadOnly}
      className="studio-pane studio-right"
      aria-label={t("workbench.studio.inspector", { defaultValue: "Inspector" })}
    >
      <button
        type="button"
        className="studio-sheet-close"
        aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
        onClick={onCloseSheet}
      >
        <X aria-hidden size={15} />
      </button>
      <StudioInspector
        tab={rightTab}
        onTab={onRightTab}
        design={design}
        frame={frame}
        theme={design.theme}
        selected={selected ?? null}
        onPropChange={(key, value) => {
          if (!selected) return;
          commitFrame(
            (spec) => setProp(spec, selected.id, key, value),
            t("workbench.studio.editProp", { defaultValue: "Edit {{key}}", key }),
            `prop:${frame.id}:${selected.id}:${key}`,
          );
        }}
        onReplaceNode={(type: TemplateNodeType) => {
          if (!selected) return;
          commitFrame(
            (spec) => updateNode(spec, selected.id, (current) => asReplacementNode(current, type)),
            t("workbench.studio.replaceLayerAction", {
              defaultValue: "Replace {{layer}} with {{candidate}}",
              layer: selected.type,
              candidate: type,
            }),
            `replace:${frame.id}:${selected.id}`,
          );
        }}
        catalogCandidates={catalog.data ?? []}
        catalogLoading={catalog.isLoading}
        catalogFailed={catalog.isError}
        pendingCatalogAssetId={pendingCatalogAssetId}
        onReplaceCatalogAsset={(componentId) => {
          if (!selected) return;
          const component = catalog.data?.find((asset) => asset.id === componentId);
          if (!component) return;
          const target = { frameId: frame.id, nodeId: selected.id };
          const request = ++catalogReplacementRequest.current;
          setPendingCatalogAssetId(component.id);
          void queryClient
            .fetchQuery({
              queryKey: wbKeys.component(component.id),
              queryFn: () => loadComponentBundle(component.id),
              staleTime: 20_000,
            })
            .then((bundle) => {
              if (request !== catalogReplacementRequest.current) return;
              const currentSelection = selectedLocation.current;
              if (
                !currentSelection ||
                currentSelection.frameId !== target.frameId ||
                currentSelection.nodeId !== target.nodeId
              ) {
                flash(
                  t("workbench.studio.catalogReplacementSelectionChanged", {
                    defaultValue:
                      "Selection changed; choose the layer again before applying this component.",
                  }),
                );
                return;
              }
              const story = bundle.stories[0];
              if (!story) {
                flash(
                  t("workbench.studio.catalogReplacementNoStory", {
                    defaultValue: "{{component}} has no published story to preview.",
                    component: component.displayName,
                  }),
                );
                return;
              }
              studio.updateFrame(
                target.frameId,
                (spec) =>
                  updateNode(spec, target.nodeId, (current) =>
                    asCatalogReplacementNode(current, component.id, story.id),
                  ),
                t("workbench.studio.replaceLayerWithComponent", {
                  defaultValue: "Replace {{layer}} with {{candidate}}",
                  layer: selected.type,
                  candidate: component.displayName,
                }),
                `replace:${target.frameId}:${target.nodeId}:${component.id}`,
              );
            })
            .catch(() =>
              flash(
                t("workbench.studio.catalogReplacementError", {
                  defaultValue: "{{component}} could not be loaded.",
                  component: component.displayName,
                }),
              ),
            )
            .finally(() => {
              if (request === catalogReplacementRequest.current) setPendingCatalogAssetId("");
            });
        }}
        coverage={coverage}
        claims={claims}
        frames={frames}
        audits={audits}
        stateGap={stateGap}
        aiScope={aiScope}
        onAiScope={onAiScope}
        onSelectFrame={selectFrame}
        onApplyProposal={studio.applyProposal}
        onApplied={(summary) =>
          flash(
            t("workbench.studio.aiApplied", {
              defaultValue: "Applied “{{summary}}” as one undoable step.",
              summary,
            }),
          )
        }
        propose={(input, signal) =>
          proposeWorkspacePatch(
            workspace.id,
            {
              ...input,
              pageId: page.id,
              activeStates: page.displays.flatMap((entry) => {
                const stateId = studio.stateByDisplay[entry.id] ?? entry.states[0]?.id ?? "";
                return stateId ? [{ displayId: entry.id, stateId }] : [];
              }),
            },
            signal,
          )
        }
      />
    </aside>
  );
}

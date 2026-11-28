import type { Dispatch, HTMLAttributes, RefObject, SetStateAction } from "react";
import { X } from "lucide-react";
import { listAllWorkspaceAssets } from "../../api/workspaces";
import type { Workspace } from "../../api/workspaces";
import type { DesignDocument, DesignFrame } from "../design-types";
import { useTranslation } from "../../i18n";
import { cx } from "../cx";
import { Tabs } from "../primitives";
import { AssetImportError } from "../studio/assets";
import { StudioAssets } from "../studio/StudioAssets";
import { StudioLayers } from "../studio/StudioLayers";
import { StudioSystem } from "../studio/StudioSystem";
import { visibleLayerRows } from "../studio/model";
import { type Coverage } from "../studio/rcl";
import type { TemplatePalette, TemplateNode } from "../templates/spec";
import { PageFields } from "./WorkspacePageFields";
import { type LoadedWorkspaceStudio } from "./workspaceStudioPageTypes";
import { type WorkspaceStudioPageActionSet } from "./workspaceStudioPageActions";

export type WorkspaceStudioLeftTab = "layers" | "system" | "assets";

interface WorkspaceStudioLeftPaneProps {
  studio: LoadedWorkspaceStudio;
  workspace: Workspace;
  design: DesignDocument;
  frames: DesignFrame[];
  frame: DesignFrame;
  rows: ReturnType<typeof visibleLayerRows>;
  selected: TemplateNode | undefined;
  coverage: Coverage;
  leftTab: WorkspaceStudioLeftTab;
  onLeftTab: (tab: WorkspaceStudioLeftTab) => void;
  leftSheetRef: RefObject<HTMLElement>;
  inertWhenReadOnly: HTMLAttributes<HTMLElement>;
  onCloseSheet: () => void;
  setCollapsed: Dispatch<SetStateAction<Set<string>>>;
  actions: WorkspaceStudioPageActionSet;
  flash: (message: string) => void;
  token: keyof TemplatePalette | null;
  onSelectToken: (token: keyof TemplatePalette | null) => void;
}

export function WorkspaceStudioLeftPane({
  studio,
  workspace,
  design,
  frames,
  frame,
  rows,
  selected,
  coverage,
  leftTab,
  onLeftTab,
  leftSheetRef,
  inertWhenReadOnly,
  onCloseSheet,
  setCollapsed,
  actions,
  flash,
  token,
  onSelectToken,
}: WorkspaceStudioLeftPaneProps) {
  const { t } = useTranslation();
  return (
    <aside
      ref={leftSheetRef}
      {...inertWhenReadOnly}
      className="studio-pane studio-left"
      aria-label={t("workbench.studio.layersPanel", {
        defaultValue: "Layers, system and assets",
      })}
    >
      <button
        type="button"
        className="studio-sheet-close"
        aria-label={t("workbench.studio.closePanel", { defaultValue: "Close panel" })}
        onClick={onCloseSheet}
      >
        <X aria-hidden size={15} />
      </button>
      <div className="studio-pane-tabs">
        <Tabs
          label={t("workbench.studio.leftTabs", { defaultValue: "Design panes" })}
          value={leftTab}
          onChange={onLeftTab}
          items={[
            {
              id: "layers",
              label: t("workbench.studio.layers", { defaultValue: "Layers" }),
              count: rows.length,
            },
            { id: "system", label: t("workbench.studio.system", { defaultValue: "System" }) },
            {
              id: "assets",
              label: t("workbench.studio.assets", { defaultValue: "Assets" }),
              count: studio.assets.length,
            },
          ]}
        />
      </div>

      <div className={cx("studio-pane-scroll", leftTab === "layers" && "is-tight")}>
        {leftTab === "layers" ? (
          <StudioLayers
            design={design}
            frame={frame}
            rows={rows}
            selectedId={selected?.id ?? null}
            coverage={coverage}
            onToggleCollapse={(id) =>
              setCollapsed((prev) => {
                const next = new Set(prev);
                if (next.has(id)) next.delete(id);
                else next.add(id);
                return next;
              })
            }
            onSelect={(id) => actions.selectNode(frame.id, id)}
            onMove={actions.moveLayer}
            onReparent={actions.reparentLayer}
            onDuplicate={actions.duplicateLayer}
            onRemove={actions.removeLayer}
            onAdd={actions.addLayer}
            onRename={(name) =>
              studio.renamePage(
                name,
                t("workbench.workspace.renamePage", { defaultValue: "Rename page" }),
              )
            }
            footer={
              <PageFields
                pageName={studio.page.name}
                route={studio.page.route}
                pageTypeId={studio.page.pageTypeId}
                stateName={studio.state?.name ?? ""}
                canRemoveState={studio.display.states.length > 1}
                onPageName={(value) =>
                  studio.updatePage(
                    { name: value },
                    t("workbench.workspace.renamePage", { defaultValue: "Rename page" }),
                    `page:${studio.page.id}:name`,
                  )
                }
                onRoute={(value) =>
                  studio.updatePage(
                    { route: value },
                    t("workbench.workspace.editRoute", { defaultValue: "Edit page route" }),
                    `page:${studio.page.id}:route`,
                  )
                }
                onPageType={(value) =>
                  studio.updatePage(
                    { pageTypeId: value },
                    t("workbench.workspace.editPageType", { defaultValue: "Edit page type" }),
                    `page:${studio.page.id}:type`,
                  )
                }
                onStateName={(value) => studio.renameState(value)}
                onRemoveState={() => studio.removeState()}
              />
            }
          />
        ) : null}

        {leftTab === "assets" ? (
          <StudioAssets
            assets={studio.assets}
            assetUrl={studio.assetUrl}
            onImport={(file, fidelity) => {
              void studio.importAsset(file, fidelity).catch((error: unknown) => {
                if (error instanceof AssetImportError) {
                  flash(
                    error.reason === "type"
                      ? t("workbench.studio.assetType", {
                          defaultValue: "Only image files can be imported as references.",
                        })
                      : t("workbench.studio.assetTooBig", {
                          defaultValue: "That image is too large. Import one under 8 MB.",
                        }),
                  );
                  return;
                }
                flash(
                  t("workbench.studio.assetFailed", {
                    defaultValue: "The image could not be uploaded. Try again.",
                  }),
                );
              });
            }}
            onPlace={actions.placeAsset}
            onRemove={(assetId) => {
              void studio.removeAsset(assetId).catch(() =>
                flash(
                  t("workbench.studio.assetRemoveFailed", {
                    defaultValue: "That reference could not be removed. Try again.",
                  }),
                ),
              );
            }}
            loadOther={async () => {
              const all = await listAllWorkspaceAssets();
              return all.filter((asset) => asset.workspaceId !== workspace.id);
            }}
          />
        ) : null}

        {leftTab === "system" ? (
          <StudioSystem
            theme={design.theme}
            frames={frames}
            activeFrame={frame}
            token={token}
            onSelectToken={onSelectToken}
            onTokenChange={(key, value) =>
              studio.updateTheme(
                { [key]: value },
                t("workbench.studio.editTokenAction", {
                  defaultValue: "Edit {{token}} token",
                  token: key,
                }),
                `theme:${key}`,
              )
            }
            onSelectNode={(id) => actions.selectNode(frame.id, id)}
            onSelectFrame={actions.selectFrame}
            coverage={coverage}
            options={design.options}
            onOptionsChange={(patch, label) => studio.updateOptions(patch, label)}
          />
        ) : null}
      </div>
    </aside>
  );
}

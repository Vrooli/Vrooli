import { Check, Image, Layers, Palette, SlidersHorizontal } from "lucide-react";
import { useTranslation } from "../../i18n";
import type { DesignDocument } from "../design-types";
import { StudioStatusBar } from "../studio/StudioStatusBar";
import type { FrameAuditMap } from "../studio/checks";
import type { Coverage } from "../studio/rcl";
import type { SavedState } from "../studio/persistence";
import type { WorkspaceStudioLeftTab } from "./WorkspaceStudioLeftPane";

interface WorkspaceStudioPageControlsProps {
  doc: DesignDocument;
  coverage: Coverage;
  audits: FrameAuditMap;
  saved: SavedState | undefined;
  dirty: boolean;
  saving: boolean;
  saveFailed: boolean;
  onRetrySave: () => void;
  openCount: number;
  readOnly: boolean;
  issueCount: number;
  zoom: number;
  onOpenLeft: (tab: WorkspaceStudioLeftTab) => void;
  onOpenRight: (tab: "inspect" | "checks") => void;
}

export function WorkspaceStudioPageControls({
  doc,
  coverage,
  audits,
  saved,
  dirty,
  saving,
  saveFailed,
  onRetrySave,
  openCount,
  readOnly,
  issueCount,
  zoom,
  onOpenLeft,
  onOpenRight,
}: WorkspaceStudioPageControlsProps) {
  const { t } = useTranslation();
  return (
    <>
      <div className="studio-mobile-bar">
        <button
          type="button"
          className="studio-mobile-btn"
          onClick={() => {
            onOpenLeft("layers");
          }}
        >
          <Layers aria-hidden size={14} />
          {t("workbench.studio.layers", { defaultValue: "Layers" })}
        </button>
        <button
          type="button"
          className="studio-mobile-btn"
          onClick={() => {
            onOpenLeft("system");
          }}
        >
          <Palette aria-hidden size={14} />
          {t("workbench.studio.system", { defaultValue: "System" })}
        </button>
        <button
          type="button"
          className="studio-mobile-btn"
          onClick={() => {
            onOpenLeft("assets");
          }}
        >
          <Image aria-hidden size={14} />
          {t("workbench.studio.assets", { defaultValue: "Assets" })}
        </button>
        <button
          type="button"
          className="studio-mobile-btn"
          onClick={() => {
            onOpenRight("inspect");
          }}
        >
          <SlidersHorizontal aria-hidden size={14} />
          {t("workbench.studio.inspect", { defaultValue: "Inspect" })}
        </button>
        <button
          type="button"
          className="studio-mobile-btn"
          onClick={() => {
            onOpenRight("checks");
          }}
        >
          <Check aria-hidden size={14} />
          {t("workbench.studio.checks", { defaultValue: "Checks" })}
        </button>
      </div>

      <StudioStatusBar
        doc={doc}
        coverage={coverage}
        audits={audits}
        saved={saved}
        dirty={dirty}
        saving={saving}
        saveFailed={saveFailed}
        onRetrySave={onRetrySave}
        openCount={openCount}
        readOnly={readOnly}
        issueCount={issueCount}
        zoom={zoom}
      />
    </>
  );
}

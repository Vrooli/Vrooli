import { Check, X } from "lucide-react";
import { useTranslation } from "../../i18n";
import type { DesignDocument, DesignFrame } from "../design-types";
import { selectors } from "../../consts/selectors";
import { useFocusTrap } from "../useFocusTrap";
import { FRAME_SIZE, frameTemplate } from "../studio/model";
import { HtmlImport } from "../studio/HtmlImport";
import { CaptureOutlinePatchDialog } from "../studio/CaptureOutlinePatchDialog";
import type { AppliedPatch } from "../studio/patch";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { LegacyCompositionPreview } from "../studio/LegacyCompositionPreview";

interface WorkspaceStudioPageOverlaysProps {
  htmlImportOpen: boolean;
  capturePatchOpen: boolean;
  onCloseHtmlImport: () => void;
  onCloseCapturePatch: () => void;
  onImportHtml: (name: string, template: TemplateNodeImport) => void;
  onApplyCapturePatch: (patch: AppliedPatch, patchId: string, summary: string) => void;
  readOnly: boolean;
  notice: string | null;
  design: DesignDocument;
  frame: DesignFrame;
  presenting: boolean;
  onClosePresentation: () => void;
  presentScale: number;
  legacyPreviewArgs?: string;
}

type TemplateNodeImport = Parameters<Parameters<typeof HtmlImport>[0]["onImport"]>[0];

export function WorkspaceStudioPageOverlays({
  htmlImportOpen,
  capturePatchOpen,
  onCloseHtmlImport,
  onCloseCapturePatch,
  onImportHtml,
  onApplyCapturePatch,
  readOnly,
  notice,
  design,
  frame,
  presenting,
  onClosePresentation,
  presentScale,
  legacyPreviewArgs,
}: WorkspaceStudioPageOverlaysProps) {
  const { t } = useTranslation();
  const presentRef = useFocusTrap<HTMLDivElement>(presenting, onClosePresentation);
  return (
    <>
      {htmlImportOpen ? (
        <HtmlImport
          onClose={onCloseHtmlImport}
          onImport={(template, name) => {
            onImportHtml(name, template);
            onCloseHtmlImport();
          }}
        />
      ) : null}

      {capturePatchOpen ? (
        <CaptureOutlinePatchDialog
          design={design}
          activeFrameId={frame.id}
          readOnly={readOnly}
          onClose={onCloseCapturePatch}
          onApply={onApplyCapturePatch}
        />
      ) : null}

      {notice ? (
        <div className="studio-toast" data-testid={selectors.workspaceStudio.notice} role="status">
          <Check aria-hidden size={13} />
          {notice}
        </div>
      ) : null}

      {presenting ? (
        <div
          ref={presentRef}
          role="dialog"
          aria-modal="true"
          aria-label={t("workbench.studio.presentLabel", { defaultValue: "Present composition" })}
          className="studio-present"
        >
          <div className="studio-present-close">
            <button
              type="button"
              className="studio-iconbtn"
              aria-label={t("workbench.studio.closePresent", {
                defaultValue: "Close presentation",
              })}
              onClick={onClosePresentation}
            >
              <X aria-hidden size={16} />
            </button>
          </div>
          <div
            className="studio-present-frame"
            style={{
              width: FRAME_SIZE[frame.device].width,
              height: legacyPreviewArgs ? "92dvh" : undefined,
              zoom: presentScale,
            }}
          >
            {legacyPreviewArgs ? (
              <LegacyCompositionPreview argsJson={legacyPreviewArgs} />
            ) : (
              <TemplateRenderer
                template={frameTemplate(design, frame)}
                spec={frame.spec}
                mode="preview"
              />
            )}
          </div>
        </div>
      ) : null}
    </>
  );
}

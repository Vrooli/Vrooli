import { FileQuestion } from "lucide-react";
import { useFileViewerText } from "../../i18n/context.js";

import { strings } from "../../i18n/strings.js";
import { PreviewActions, PreviewMetaLine } from "./shared.js";
import type { PreviewRendererProps } from "../types.js";

// UnsupportedPreview is the renderer for files with no dedicated viewer. It
// stays useful: it shows metadata plus download/open/copy-path affordances.
export function UnsupportedPreview({ model }: PreviewRendererProps) {
  const { t } = useFileViewerText();
  return (
    <div className="flex h-full items-center justify-center p-6" data-testid="file-preview-unsupported">
      <div className="flex max-w-md flex-col items-center gap-3 text-center">
        <div className="rounded-full border border-fv-default bg-fv-surface-input p-3 text-fv-text-muted">
          <FileQuestion className="h-6 w-6" />
        </div>
        <h3 className="text-sm font-semibold text-fv-text-primary">{t(strings.viewer.unsupportedTitle)}</h3>
        <p className="text-sm text-fv-text-muted">{t(strings.viewer.unsupportedBody)}</p>
        <PreviewMetaLine model={model} />
        <PreviewActions model={model} className="justify-center" />
      </div>
    </div>
  );
}

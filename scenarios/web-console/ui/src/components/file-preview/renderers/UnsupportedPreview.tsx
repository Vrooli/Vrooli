import { FileQuestion } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { PreviewActions, PreviewMetaLine } from "./shared";
import type { PreviewRendererProps } from "../types";

// UnsupportedPreview is the renderer for files with no dedicated viewer. It
// stays useful: it shows metadata plus download/open/copy-path affordances.
export function UnsupportedPreview({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  return (
    <div className="flex h-full items-center justify-center p-6" data-testid="file-preview-unsupported">
      <div className="flex max-w-md flex-col items-center gap-3 text-center">
        <div className="rounded-full border border-wc-default bg-wc-surface-input p-3 text-wc-text-muted">
          <FileQuestion className="h-6 w-6" />
        </div>
        <h3 className="text-sm font-semibold text-wc-text-primary">{t(strings.messagesFileViewer.unsupportedTitle)}</h3>
        <p className="text-sm text-wc-text-muted">{t(strings.messagesFileViewer.unsupportedBody)}</p>
        <PreviewMetaLine model={model} />
        <PreviewActions model={model} className="justify-center" />
      </div>
    </div>
  );
}

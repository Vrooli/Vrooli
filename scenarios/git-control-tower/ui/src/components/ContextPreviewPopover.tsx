import { X } from "lucide-react";
import { Popover } from "./ui/popover";
import { MarkdownPreview } from "./MarkdownPreview";
import { buildCaptureScreenshotUrl } from "../lib/api";
import type { AgentContextItem } from "../lib/api";

interface ContextPreviewPopoverProps {
  item: AgentContextItem;
  scenarioSlug: string;
  onRemove: (id: string) => void;
}

/** Parse captureId and filename from a screenshot context item id. */
function parseScreenshotId(id: string): { captureId: string; filename: string } | null {
  const match = id.match(/^screenshot-(.+?)-([^-]+\.\w+)$/);
  if (!match?.[1] || !match[2]) return null;
  return { captureId: match[1], filename: match[2] };
}

export function ContextPreviewPopover({ item, scenarioSlug, onRemove }: ContextPreviewPopoverProps) {
  const screenshotInfo = item.kind === "screenshot" ? parseScreenshotId(item.id) : null;

  return (
    <Popover
      direction="up"
      align="start"
      trigger={
        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-[11px] text-slate-300 cursor-pointer hover:border-slate-600 hover:bg-slate-750 transition-colors">
          {item.label}
          <span
            role="button"
            className="text-slate-500 hover:text-slate-300"
            onClick={(e) => {
              e.stopPropagation();
              onRemove(item.id);
            }}
          >
            <X className="h-3 w-3" />
          </span>
        </span>
      }
    >
      <div className="w-96 max-h-80 overflow-y-auto text-sm">
        {screenshotInfo && (
          <div className="p-2 border-b border-slate-700">
            <img
              src={buildCaptureScreenshotUrl(screenshotInfo.captureId, scenarioSlug, screenshotInfo.filename)}
              alt={item.label}
              className="w-full rounded object-contain bg-slate-950"
              loading="lazy"
            />
          </div>
        )}
        <MarkdownPreview content={item.markdown} />
      </div>
    </Popover>
  );
}

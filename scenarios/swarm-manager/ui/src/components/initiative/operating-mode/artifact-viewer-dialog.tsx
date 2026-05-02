/**
 * ArtifactViewerDialog
 *
 * Read-only viewer for a single OperatingMode artifact. Renders markdown
 * via the shared `renderMarkdown` helper for `text/markdown`; everything
 * else falls back to a plain `<pre>` so the source is still readable.
 *
 * The Download button uses an in-memory Blob + URL.createObjectURL so the
 * client never re-fetches the artifact — content is already in props from
 * the workspace response. The artifact's `path` (basename) is used as the
 * download filename.
 */

import { useEffect, useMemo } from "react";
import { Download } from "lucide-react";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import { renderMarkdown } from "../../../lib/render-markdown";
import { formatRelativeTime } from "../../../lib";
import type { OperatingModeArtifactSnapshot } from "../../../types/operating-mode";

export interface ArtifactViewerDialogProps {
  artifact: OperatingModeArtifactSnapshot | null;
  isOpen: boolean;
  onClose: () => void;
}

function basename(path: string): string {
  const parts = path.split("/");
  return parts[parts.length - 1] || path;
}

export function ArtifactViewerDialog({ artifact, isOpen, onClose }: ArtifactViewerDialogProps) {
  const blobURL = useMemo(() => {
    if (!artifact?.content) return null;
    const blob = new Blob([artifact.content], {
      type: artifact.contentType || "text/plain",
    });
    return URL.createObjectURL(blob);
  }, [artifact?.content, artifact?.contentType]);

  useEffect(() => {
    return () => {
      if (blobURL) URL.revokeObjectURL(blobURL);
    };
  }, [blobURL]);

  if (!artifact) return null;

  const isMarkdown = artifact.contentType === "text/markdown";
  const downloadName = basename(artifact.path);

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={artifact.path}
      maxWidth="max-w-3xl"
      testId={selectors.initiativeDetails.artifactViewerDialog}
    >
      <div className="space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-[11px] text-slate-500">
          {artifact.contentType && (
            <span className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5">
              {artifact.contentType}
            </span>
          )}
          {typeof artifact.sizeBytes === "number" && (
            <span>{artifact.sizeBytes} bytes</span>
          )}
          {artifact.required && (
            <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-amber-300">
              required
            </span>
          )}
          {artifact.updatedAt && <span>{formatRelativeTime(artifact.updatedAt)}</span>}
        </div>

        {artifact.content ? (
          isMarkdown ? (
            <div
              className="prose prose-invert max-w-none rounded-md border border-slate-800 bg-slate-900/40 p-4"
              dangerouslySetInnerHTML={{ __html: renderMarkdown(artifact.content) }}
            />
          ) : (
            <pre className="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-md border border-slate-800 bg-slate-950/70 p-3 text-xs leading-relaxed text-slate-300">
              {artifact.content}
            </pre>
          )
        ) : (
          <p className="rounded-md border border-slate-800 bg-slate-900/40 p-4 text-sm italic text-slate-500">
            Artifact not created yet.
          </p>
        )}

        <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
          {blobURL ? (
            <a
              href={blobURL}
              download={downloadName}
              className="inline-flex items-center justify-center rounded-full border border-slate-300/40 px-3 py-1.5 text-xs font-medium text-slate-50 transition-colors hover:bg-slate-900/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/40"
              data-testid={selectors.initiativeDetails.artifactDownload}
            >
              <Download className="mr-1.5 h-3.5 w-3.5" />
              Download
            </a>
          ) : (
            <span />
          )}
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

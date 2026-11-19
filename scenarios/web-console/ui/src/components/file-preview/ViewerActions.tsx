import type { ViewerActionContext } from "@vrooli/file-viewer";
import { Send } from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import { scriptRunPlan } from "../../lib/fileExecution";

interface ViewerActionsProps {
  context: ViewerActionContext;
  /**
   * Hand this file to another session in the group.
   *
   * This is the operator's existing habit — open the plan from the Messages
   * view, read it, then pass it on — so it is where the handoff earns the
   * most. The resolved path is the payload; the viewer does not classify it.
   */
  onHandoff?: (resolvedPath: string) => void;
  onRun?: (resolvedPath: string, text: string) => void;
}

/**
 * Web Console's own actions in the shared file viewer's header: hand a file to
 * another session, or run a script. Both belong to sessions, so they live here
 * and reach the viewer through its header-actions slot.
 */
export function ViewerActions({ context, onHandoff, onRun }: ViewerActionsProps) {
  const { t } = useTranslation();
  const { path, model, text } = context;
  // The handoff sits beside Back, and only for a file with a resolved path:
  // there is nothing to hand over while the preview is still resolving, and a
  // directory listing is not a payload the operator meant to send.
  const canHandoff = Boolean(onHandoff && !model?.archiveMember && path && model?.kind !== "directory");
  const canRun = Boolean(onRun && !model?.archiveMember && model?.kind === "code" && scriptRunPlan(path, text?.content ?? ""));
  if (!canHandoff && !canRun) return null;
  return (
    <>
      {canHandoff && (
        <button
          type="button"
          onClick={() => onHandoff?.(path)}
          data-testid="handoff-file-viewer"
          aria-label={t(strings.fileViewerActions.handOffTitle)}
          title={t(strings.fileViewerActions.handOffTitle)}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-raised hover:text-wc-text-primary"
        >
          <Send className="h-3.5 w-3.5" />
          {t(strings.fileViewerActions.handOff)}
        </button>
      )}
      {canRun && (
        <button
          type="button"
          onClick={() => onRun?.(path, text?.content ?? "")}
          data-testid="run-file-viewer"
          className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-wc-accent px-2 py-1.5 text-xs font-medium text-white"
        >
          {t(strings.fileViewerActions.run)}
        </button>
      )}
    </>
  );
}

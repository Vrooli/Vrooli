import { ViewerButton, type ViewerActionContext } from "@vrooli/file-viewer";
import { Play, Send } from "lucide-react";
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
        <ViewerButton
          type="button"
          variant="secondary"
          onClick={() => onHandoff?.(path)}
          data-testid="handoff-file-viewer"
          aria-label={t(strings.fileViewerActions.handOffTitle)}
          title={t(strings.fileViewerActions.handOffTitle)}
        >
          <Send className="h-4 w-4" />
          <span className="hidden sm:inline">{t(strings.fileViewerActions.handOff)}</span>
        </ViewerButton>
      )}
      {canRun && (
        <ViewerButton
          type="button"
          variant="primary"
          onClick={() => onRun?.(path, text?.content ?? "")}
          data-testid="run-file-viewer"
          aria-label={t(strings.fileViewerActions.run)}
        >
          <Play className="h-4 w-4" />
          <span className="hidden sm:inline">{t(strings.fileViewerActions.run)}</span>
        </ViewerButton>
      )}
    </>
  );
}

import { useState } from "react";
import { captureWorkspaceState, getWorkspace, WorkspaceCaptureError } from "../../api/workspaces";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";
import type { WorkspaceStudioPageTranslate } from "./workspaceStudioPageActions";

function newCaptureAttemptId(): string {
  const cryptoAPI = globalThis.crypto as Partial<Crypto>;
  if (typeof cryptoAPI.randomUUID === "function") return cryptoAPI.randomUUID();
  const bytes = new Uint8Array(16);
  if (typeof cryptoAPI.getRandomValues === "function") cryptoAPI.getRandomValues(bytes);
  else
    for (let index = 0; index < bytes.length; index += 1)
      bytes[index] = Math.floor(Math.random() * 256);
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

interface WorkspaceStudioCaptureInput {
  workspaceId: string;
  studio: ReturnType<typeof useWorkspaceStudio>;
  translate: WorkspaceStudioPageTranslate;
  flash: (message: string) => void;
}

export function useWorkspaceStudioCapture({
  workspaceId,
  studio,
  translate: t,
  flash,
}: WorkspaceStudioCaptureInput) {
  const [capturingState, setCapturingState] = useState(false);

  const captureSelectedState = async () => {
    if (
      !studio.workspace ||
      !studio.page ||
      !studio.display ||
      !studio.state ||
      studio.dirty ||
      studio.saving ||
      studio.readOnly ||
      capturingState
    )
      return;
    const selected = {
      expectedRevision: studio.workspace.revision,
      pageId: studio.page.id,
      displayId: studio.display.id,
      stateId: studio.state.id,
    };
    const attemptKey = `rcl-capture-attempt:${workspaceId}:${selected.stateId}:${selected.expectedRevision}`;
    let attemptId = sessionStorage.getItem(attemptKey);
    if (!attemptId) {
      attemptId = newCaptureAttemptId();
      sessionStorage.setItem(attemptKey, attemptId);
    }
    setCapturingState(true);
    try {
      await captureWorkspaceState(workspaceId, { ...selected, attemptId });
      const latest = await getWorkspace(workspaceId);
      if (!latest) throw new Error("Captured workspace could not be reloaded.");
      await studio.syncRemote(latest);
      sessionStorage.removeItem(attemptKey);
      flash(
        t("workbench.workspace.captureComplete", {
          defaultValue: "Workspace state captured by BAS.",
        }),
      );
    } catch (cause) {
      if (cause instanceof WorkspaceCaptureError && cause.attemptState === "failed") {
        // A terminal BAS failure is immutable for this idempotency token. Keep
        // its evidence at the server and make the next explicit click a new attempt.
        sessionStorage.removeItem(attemptKey);
      }
      flash(
        cause instanceof Error
          ? cause.message
          : t("workbench.workspace.captureFailed", {
              defaultValue: "Workspace state capture could not be completed.",
            }),
      );
    } finally {
      setCapturingState(false);
    }
  };

  return { capturingState, captureSelectedState };
}

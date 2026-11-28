import { useCallback, type MutableRefObject } from "react";

import type { Workspace } from "../../api/workspaces";
import { docSignature } from "../studio/model";
import { activeStateOf, pageDocument, writeStateSpec } from "../workspaces/spec";
import { applyPatch, PatchError } from "./patch";
import type { ProposalApplication, StudioProposal } from "./proposal";

type ApplyWorkspaceChange = (
  label: string,
  mergeKey: string | undefined,
  change: (draft: Workspace) => void,
) => void;

interface WorkspaceProposalOptions {
  workspaceRef: MutableRefObject<Workspace | null>;
  pageIdRef: MutableRefObject<string>;
  statesRef: MutableRefObject<Record<string, string>>;
  apply: ApplyWorkspaceChange;
}

/** Applies proposals only while their design and visible composition still match. */
export function useWorkspaceProposal({
  workspaceRef,
  pageIdRef,
  statesRef,
  apply,
}: WorkspaceProposalOptions): (proposal: StudioProposal) => ProposalApplication {
  return useCallback(
    (proposal: StudioProposal): ProposalApplication => {
      const current = workspaceRef.current;
      const activePage =
        current?.pages.find((entry) => entry.id === pageIdRef.current) ?? current?.pages[0];
      if (!current || !activePage) {
        return { ok: false, reason: "invalid", detail: "There is no active page." };
      }
      const doc = pageDocument(current, activePage, statesRef.current);
      // A revision bump from autosave does not change the composition, so the
      // visible document signature is the right stale proposal guard.
      if (proposal.designId !== current.id || proposal.signature !== docSignature(doc)) {
        return {
          ok: false,
          reason: "stale",
          detail: "The workspace changed after this proposal was made.",
        };
      }

      let applied;
      try {
        applied = applyPatch(doc, proposal.operations);
      } catch (error) {
        return {
          ok: false,
          reason: "invalid",
          detail: error instanceof PatchError ? error.reason : "The patch could not be applied.",
        };
      }

      apply(`AI · ${proposal.summary}`, undefined, (draft) => {
        const targetPage = draft.pages.find((entry) => entry.id === activePage.id);
        if (!targetPage) return;
        for (const frame of applied.doc.frames) {
          const display = targetPage.displays.find((entry) => entry.id === frame.id);
          if (!display) continue;
          const state = activeStateOf(display, statesRef.current);
          if (state) writeStateSpec(state, frame.spec);
        }
        // Text and layer edits do not own the theme; only setTheme does.
        if (proposal.operations.some((operation) => operation.kind === "setTheme")) {
          draft.themeJson = JSON.stringify(applied.doc.theme);
        }
      });
      return { ok: true };
    },
    [apply, pageIdRef, statesRef, workspaceRef],
  );
}

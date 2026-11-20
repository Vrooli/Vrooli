/**
 * The workspace studio's AI transport.
 *
 * It is a typed Connect call, not a second REST surface, and the server never
 * writes: it validates a proposed patch against the page it read and returns it
 * as opaque JSON. The studio applies it as one undoable command.
 */
import { Code, ConnectError } from "@connectrpc/connect";

import { workspacesClient } from "../../api/workspaces";
import {
  DesignProposalError,
  type DesignPatchOperation,
  type DesignProposal,
  type DesignProposalErrorCode,
  type ProposeInput,
} from "./proposal";

export interface WorkspaceProposeInput extends ProposeInput {
  pageId: string;
  /** The state on screen per display, so the server reads the same frames. */
  activeStates: Array<{ displayId: string; stateId: string }>;
}

function codeFor(error: ConnectError): DesignProposalErrorCode {
  switch (error.code) {
    case Code.Aborted:
    case Code.AlreadyExists:
      return "conflict";
    case Code.Unavailable:
      return "ai_unavailable";
    case Code.FailedPrecondition:
      return "invalid_proposal";
    case Code.InvalidArgument:
      return "invalid_argument";
    default:
      return "ai_failed";
  }
}

export async function proposeWorkspacePatch(
  workspaceId: string,
  input: WorkspaceProposeInput,
  signal?: AbortSignal,
): Promise<DesignProposal> {
  let response;
  try {
    response = await workspacesClient.assistWorkspace(
      {
        id: workspaceId,
        baseRevision: input.revision,
        pageId: input.pageId,
        activeStates: input.activeStates,
        prompt: input.prompt,
        scope: {
          kind: input.scope.kind,
          displayId: input.scope.frameId ?? "",
          nodeId: input.scope.nodeId ?? "",
        },
        idempotencyKey: input.idempotencyKey,
      },
      { signal },
    );
  } catch (cause) {
    if (signal?.aborted) {
      throw new DesignProposalError("ai_unavailable", 0, "The request was canceled.");
    }
    const error = ConnectError.from(cause);
    throw new DesignProposalError(codeFor(error), 0, error.rawMessage);
  }

  let parsed: { summary?: string; operations?: DesignPatchOperation[] };
  try {
    parsed = JSON.parse(response.proposalJson) as {
      summary?: string;
      operations?: DesignPatchOperation[];
    };
  } catch {
    throw new DesignProposalError("invalid_proposal", 0, "The gateway returned unreadable JSON.");
  }
  if (!Array.isArray(parsed.operations) || parsed.operations.length === 0) {
    throw new DesignProposalError("invalid_proposal", 0, "The gateway returned no patch.");
  }
  return {
    summary: parsed.summary ?? "",
    operations: parsed.operations,
    revision: response.revision,
    operationId: response.operation?.id ?? "",
    state: response.operation?.state ?? "completed",
    provider: response.operation?.provider,
    model: response.operation?.model,
  };
}

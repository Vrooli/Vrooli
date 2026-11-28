/**
 * AI proposal shapes shared by both studio surfaces.
 *
 * The wire shape is deliberately flat (one `kind` plus optional fields)
 * because that is what models follow reliably; the studio validates every field
 * before it touches the document. The server validates the same shape against
 * the document it read, so an unusable patch is refused before it reaches here.
 */
import type { TemplateNodeType } from "../templates/spec";

export interface DesignPatchOperation {
  kind: "setText" | "setProp" | "addLayer" | "removeLayer" | "moveLayer" | "setTheme";
  nodeId?: string;
  parentId?: string;
  key?: string;
  value?: string | number | boolean;
  text?: string;
  node?: DesignNewLayer;
  index?: number;
}

export interface DesignNewLayer {
  type: TemplateNodeType;
  props: Record<string, string | number | boolean>;
  children?: DesignNewLayer[];
}

export interface DesignProposal {
  summary: string;
  operations: DesignPatchOperation[];
  /** The revision this proposal was computed against. */
  revision: number;
  operationId: string;
  state: string;
  provider?: string;
  model?: string;
  inputTokens?: number;
  outputTokens?: number;
}

export type DesignProposalErrorCode =
  | "ai_unavailable"
  | "ai_failed"
  | "invalid_proposal"
  | "invalid_argument"
  | "conflict"
  | "unknown";

export class DesignProposalError extends Error {
  readonly code: DesignProposalErrorCode;
  readonly status: number;

  constructor(code: DesignProposalErrorCode, status: number, message: string) {
    super(message);
    this.name = "DesignProposalError";
    this.code = code;
    this.status = status;
  }
}

export const PROPOSAL_CODES: readonly DesignProposalErrorCode[] = [
  "ai_unavailable",
  "ai_failed",
  "invalid_proposal",
  "invalid_argument",
  "conflict",
];

export interface ProposeInput {
  prompt: string;
  scope: { kind: "design" | "frame" | "layer"; frameId?: string; nodeId?: string };
  revision: number;
  idempotencyKey: string;
}

export type ProposalTransport = (
  input: ProposeInput,
  signal?: AbortSignal,
) => Promise<DesignProposal>;

/**
 * A proposal plus the two facts the server cannot know: which composition it
 * was made for, and the exact document it saw. Applying is refused when either
 * changed, so a slow model response can never patch a different composition.
 */
export interface StudioProposal extends DesignProposal {
  designId: string;
  signature: string;
}

export type ProposalApplication =
  | { ok: true }
  | { ok: false; reason: "stale" | "invalid"; detail: string };

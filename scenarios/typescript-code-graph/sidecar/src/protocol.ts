// IPC protocol types for the typescript-code-graph Node sidecar.
//
// Wire format: line-delimited JSON over stdio. Every request carries a
// `request_id` (UUID v4 generated on the Go side); every response echoes it.
//
// All JSON keys are snake_case to match proto field names exactly. The Go
// supervisor unmarshals these directly into proto-compatible Go structs.

export const PROTOCOL_VERSION = 1;

// --- Error kinds --------------------------------------------------------------

export type ErrorKind =
  | "parse_failure"
  | "no_tsconfig_found"
  | "multiple_tsconfig_files"
  | "workspace_unsupported"
  | "path_unreadable"
  | "cancelled"
  | "internal";

// --- Request envelopes --------------------------------------------------------

export interface HandshakeRequest {
  type: "handshake";
  request_id: string;
  protocol_version: number;
}

export interface HeartbeatRequest {
  type: "heartbeat";
  request_id: string;
}

export interface ExtractRequest {
  type: "extract";
  request_id: string;
  scenario_path: string;
}

export interface RewriteOperation {
  // Exactly one of `file_move` / `import_rewrite` is set.
  file_move?: {
    from_path: string;
    to_path: string;
  };
  import_rewrite?: {
    old_path: string;
    new_path: string;
  };
}

export interface RewriteApplyRequest {
  type: "rewrite_apply";
  request_id: string;
  scenario_path: string;
  operations: RewriteOperation[];
}

export interface CancelRequest {
  type: "cancel";
  request_id: string;
}

export interface ShutdownRequest {
  type: "shutdown";
  request_id?: string;
}

export type Request =
  | HandshakeRequest
  | HeartbeatRequest
  | ExtractRequest
  | RewriteApplyRequest
  | CancelRequest
  | ShutdownRequest;

// --- Response envelopes -------------------------------------------------------

export interface HandshakeResponse {
  type: "handshake";
  request_id: string;
  protocol_version: number;
  sidecar_version: string;
  ts_morph_version: string;
}

export interface HeartbeatResponse {
  type: "heartbeat";
  request_id: string;
}

// Mirrors common.v1.CodeGraph wire shape (snake_case keys).
export interface CodeGraphNode {
  id: string;
  // Numeric NodeKind from common/v1/code_graph.proto. FILE=1, PACKAGE=2,
  // MODULE=3. TS-specific kinds (200-209) ride on attributes["kind"].
  kind: number;
  name: string;
  path: string;
  attributes: Record<string, string>;
  leading_comments: string[];
}

export interface CodeGraphEdge {
  id: string;
  // EdgeKind: IMPORT=1, INTRA_PACKAGE_REF=2, RE_EXPORT=3.
  kind: number;
  from_node_id: string;
  to_node_id: string;
  attributes: Record<string, string>;
}

export interface CodeGraph {
  nodes: CodeGraphNode[];
  edges: CodeGraphEdge[];
}

export interface CodeGraphWarning {
  // CodeGraphWarningKind: PARSE_ERROR=1, UNRESOLVED_IMPORT=2,
  // TYPE_CHECK_FAILURE=3, AMBIGUOUS_DECLARATION=4.
  kind: number;
  file: string;
  message: string;
}

export interface ExtractResponse {
  type: "extract";
  request_id: string;
  graph: CodeGraph;
  warnings: CodeGraphWarning[];
}

export interface OperationResult {
  ok: boolean;
  // Number of files touched (move=1, import_rewrite=N).
  touched: number;
  message: string;
}

export interface RewriteApplyResponse {
  type: "rewrite_apply";
  request_id: string;
  results: OperationResult[];
}

export interface ErrorResponse {
  type: "error";
  request_id: string;
  kind: ErrorKind;
  message: string;
}

export type Response =
  | HandshakeResponse
  | HeartbeatResponse
  | ExtractResponse
  | RewriteApplyResponse
  | ErrorResponse;

// --- Type guards --------------------------------------------------------------

export function isRequest(value: unknown): value is Request {
  if (typeof value !== "object" || value === null) return false;
  const v = value as { type?: unknown };
  return (
    v.type === "handshake" ||
    v.type === "heartbeat" ||
    v.type === "extract" ||
    v.type === "rewrite_apply" ||
    v.type === "cancel" ||
    v.type === "shutdown"
  );
}

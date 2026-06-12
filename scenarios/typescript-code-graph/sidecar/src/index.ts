// Sidecar entrypoint. Reads line-delimited JSON requests from stdin,
// dispatches to handlers, writes line-delimited JSON responses to stdout.
// Logs ALL diagnostics to stderr — stdout is reserved for the framer.

import * as readline from "node:readline";

import { extract, NoTsConfigError, MultipleTsConfigError, WorkspaceUnsupportedError, PathUnreadableError, ParseFailureError } from "./extract.js";
import { handleHandshake } from "./handshake.js";
import { withPathLock } from "./lock.js";
import { logger } from "./logger.js";
import { applyRewrite } from "./rewrite.js";
import { parseMessage, writeMessage } from "./framing.js";
import type {
  CancelRequest,
  ErrorKind,
  ErrorResponse,
  ExtractRequest,
  HandshakeRequest,
  HeartbeatRequest,
  Request,
  RewriteApplyRequest,
  ShutdownRequest,
} from "./protocol.js";
import { isRequest } from "./protocol.js";

interface InFlightEntry {
  cancelled: boolean;
}

const inFlight = new Map<string, InFlightEntry>();
let shuttingDown = false;
let activeWork = 0;

function errResp(request_id: string, kind: ErrorKind, message: string): ErrorResponse {
  return { type: "error", request_id, kind, message };
}

function classifyKnownError(err: unknown): ErrorKind {
  if (err instanceof NoTsConfigError) return "no_tsconfig_found";
  if (err instanceof MultipleTsConfigError) return "multiple_tsconfig_files";
  if (err instanceof WorkspaceUnsupportedError) return "workspace_unsupported";
  if (err instanceof PathUnreadableError) return "path_unreadable";
  if (err instanceof ParseFailureError) return "parse_failure";
  return "internal";
}

async function handleExtract(req: ExtractRequest): Promise<void> {
  const entry: InFlightEntry = { cancelled: false };
  inFlight.set(req.request_id, entry);
  activeWork++;
  try {
    const result = await withPathLock(req.project_path, async () => {
      return extract({ projectPath: req.project_path });
    });
    if (entry.cancelled) {
      logger.debug("discarding result of cancelled extract", { request_id: req.request_id });
      return;
    }
    writeMessage({
      type: "extract",
      request_id: req.request_id,
      graph: result.graph,
      warnings: result.warnings,
    });
  } catch (err) {
    if (entry.cancelled) return;
    const kind = classifyKnownError(err);
    writeMessage(errResp(req.request_id, kind, (err as Error).message));
  } finally {
    inFlight.delete(req.request_id);
    activeWork--;
  }
}

async function handleRewriteApply(req: RewriteApplyRequest): Promise<void> {
  const entry: InFlightEntry = { cancelled: false };
  inFlight.set(req.request_id, entry);
  activeWork++;
  try {
    const results = await withPathLock(req.project_path, async () => {
      return applyRewrite({
        projectPath: req.project_path,
        operations: req.operations,
      });
    });
    if (entry.cancelled) {
      logger.debug("discarding result of cancelled rewrite_apply", { request_id: req.request_id });
      return;
    }
    writeMessage({
      type: "rewrite_apply",
      request_id: req.request_id,
      results,
    });
  } catch (err) {
    if (entry.cancelled) return;
    writeMessage(errResp(req.request_id, "internal", (err as Error).message));
  } finally {
    inFlight.delete(req.request_id);
    activeWork--;
  }
}

function handleHeartbeat(req: HeartbeatRequest): void {
  writeMessage({ type: "heartbeat", request_id: req.request_id });
}

function handleCancel(req: CancelRequest): void {
  const entry = inFlight.get(req.request_id);
  if (entry) {
    entry.cancelled = true;
    logger.debug("marked cancelled", { request_id: req.request_id });
  }
}

async function handleShutdown(_req: ShutdownRequest): Promise<void> {
  shuttingDown = true;
  // Bounded drain: wait up to 5s for in-flight work.
  const deadline = Date.now() + 5000;
  while (activeWork > 0 && Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 25));
  }
  logger.info("sidecar shutting down", { active_remaining: activeWork });
  process.exit(0);
}

async function dispatch(msg: Request): Promise<void> {
  switch (msg.type) {
    case "handshake":
      writeMessage(handleHandshake(msg as HandshakeRequest));
      return;
    case "heartbeat":
      handleHeartbeat(msg as HeartbeatRequest);
      return;
    case "extract":
      await handleExtract(msg as ExtractRequest);
      return;
    case "rewrite_apply":
      await handleRewriteApply(msg as RewriteApplyRequest);
      return;
    case "cancel":
      handleCancel(msg as CancelRequest);
      return;
    case "shutdown":
      await handleShutdown(msg as ShutdownRequest);
      return;
  }
}

export function startStdioLoop(): void {
  const rl = readline.createInterface({ input: process.stdin });

  rl.on("line", (line) => {
    if (shuttingDown) return;
    const trimmed = line.trim();
    if (!trimmed) return;
    let parsed: unknown;
    try {
      parsed = parseMessage(trimmed);
    } catch (err) {
      logger.error("malformed JSON request", { err: (err as Error).message });
      writeMessage(errResp("", "internal", `malformed request: ${(err as Error).message}`));
      return;
    }
    if (!isRequest(parsed)) {
      const reqId =
        typeof parsed === "object" && parsed !== null && "request_id" in (parsed as Record<string, unknown>)
          ? String((parsed as { request_id?: unknown }).request_id ?? "")
          : "";
      writeMessage(errResp(reqId, "internal", "unknown message type"));
      return;
    }
    // Fire-and-forget; handlers write their own responses.
    void dispatch(parsed).catch((err: unknown) => {
      const reqId = (parsed as { request_id?: string }).request_id ?? "";
      writeMessage(errResp(reqId, "internal", (err as Error).message));
    });
  });

  rl.on("close", () => {
    logger.info("stdin closed; exiting");
    process.exit(0);
  });

  process.on("uncaughtException", (err) => {
    logger.error("uncaughtException", { err: err.message, stack: err.stack });
    try {
      writeMessage(errResp("", "internal", `uncaughtException: ${err.message}`));
    } catch {
      // ignore — we're going down
    }
    process.exit(1);
  });

  process.on("unhandledRejection", (reason) => {
    logger.error("unhandledRejection", { reason: String(reason) });
    process.exit(1);
  });
}

// Only auto-start the loop when invoked as the entrypoint (not when imported
// by tests). esbuild preserves import.meta.url so this works in bundled form.
const invokedDirect =
  process.argv[1] !== undefined &&
  (import.meta.url === `file://${process.argv[1]}` ||
    import.meta.url.endsWith("/dist/index.js"));

if (invokedDirect) {
  startStdioLoop();
}

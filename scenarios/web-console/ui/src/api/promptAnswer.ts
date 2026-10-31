import { ConnectError, createClient } from "@connectrpc/connect";
import { TerminalService } from "@vrooli/proto-types/web-console/v1/terminal/terminal_pb";
import { transport } from "./client";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#maturity-levels-per-harness

const terminalClient = createClient(TerminalService, transport);

export interface PromptAnswerRequest {
  /** The option to choose; omitted for a cancel. */
  optionKey?: string;
  /** The hash of the prompt being answered, from the session's activity. */
  promptHash: string;
  cancel: boolean;
}

export interface PromptAnswerResult {
  /** "keystrokes" (typed into the terminal) or "harness_api" (the harness's reply API). */
  delivery: string;
  /** The chosen option's label; empty for a cancel. */
  answer: string;
}

/**
 * Answer the prompt a session's agent is showing. Rejects with the server's
 * reason when it refuses (the prompt changed, answering is off, ...).
 */
export async function answerPrompt(sessionId: string, request: PromptAnswerRequest): Promise<PromptAnswerResult> {
  try {
    const response = await terminalClient.answerPrompt({
      sessionId,
      optionKey: request.optionKey ?? "",
      promptHash: request.promptHash,
      cancel: request.cancel,
    });
    return { delivery: response.delivery, answer: response.answer };
  } catch (error) {
    throw new Error(error instanceof ConnectError ? error.rawMessage : String(error));
  }
}

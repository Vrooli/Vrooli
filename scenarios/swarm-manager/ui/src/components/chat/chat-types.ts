import type { ReactNode } from "react";
import type { AgentSessionContextItem } from "../../types";
import type { ChatDensity } from "../../lib/chat-density";

export type ChatRole = "user" | "assistant" | "system";

export type ChatAccent = "cyan" | "violet" | "slate";

/** Local-only delivery state for a message the client has optimistically
 * rendered. Absent on every message the server has confirmed. */
export type ChatDeliveryState = "sending" | "failed";

export interface ChatMessageView {
  id: string;
  role: ChatRole;
  content: string;
  createdAt?: string;
  attachmentIds?: string[];
  context?: AgentSessionContextItem[];
  /**
   * Set while a locally-composed message is in flight or after it failed. The
   * server never sets this: a confirmed message simply has no delivery state.
   */
  delivery?: ChatDeliveryState;
}

export type ChatMessageRenderSlot = (message: ChatMessageView) => ReactNode;

export type { ChatDensity };

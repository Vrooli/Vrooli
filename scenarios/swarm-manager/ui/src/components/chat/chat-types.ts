import type { ReactNode } from "react";
import type { AgentSessionContextItem } from "../../types";

export type ChatRole = "user" | "assistant" | "system";

export type ChatAccent = "cyan" | "violet" | "slate";

export interface ChatMessageView {
  id: string;
  role: ChatRole;
  content: string;
  createdAt?: string;
  attachmentIds?: string[];
  context?: AgentSessionContextItem[];
}

export type ChatMessageRenderSlot = (message: ChatMessageView) => ReactNode;

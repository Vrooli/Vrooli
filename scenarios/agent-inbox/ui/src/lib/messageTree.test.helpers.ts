/**
 * Shared test helpers for messageTree tests
 */

import type { Message } from "./api";

// Helper to create test messages
export function createMessage(
  id: string,
  overrides: Partial<Message> = {}
): Message {
  return {
    id,
    chat_id: "chat-1",
    role: "user",
    content: `Message ${id}`,
    sibling_index: 0,
    created_at: new Date().toISOString(),
    ...overrides,
  };
}

// Helper to create a simple linear conversation
export function createLinearConversation(): Message[] {
  return [
    createMessage("msg-1", { role: "user", content: "Hello", created_at: "2025-01-01T00:00:00Z" }),
    createMessage("msg-2", { role: "assistant", content: "Hi!", parent_message_id: "msg-1", created_at: "2025-01-01T00:01:00Z" }),
    createMessage("msg-3", { role: "user", content: "How are you?", parent_message_id: "msg-2", created_at: "2025-01-01T00:02:00Z" }),
    createMessage("msg-4", { role: "assistant", content: "I'm good!", parent_message_id: "msg-3", created_at: "2025-01-01T00:03:00Z" }),
  ];
}

// Helper to create a branching conversation with regenerated responses
export function createBranchingConversation(): Message[] {
  return [
    createMessage("msg-1", { role: "user", content: "Hello" }),
    createMessage("msg-2a", { role: "assistant", content: "Response A", parent_message_id: "msg-1", sibling_index: 0 }),
    createMessage("msg-2b", { role: "assistant", content: "Response B", parent_message_id: "msg-1", sibling_index: 1 }),
    createMessage("msg-2c", { role: "assistant", content: "Response C", parent_message_id: "msg-1", sibling_index: 2 }),
    createMessage("msg-3", { role: "user", content: "Tell me more", parent_message_id: "msg-2b" }),
    createMessage("msg-4", { role: "assistant", content: "More details...", parent_message_id: "msg-3" }),
  ];
}

// Helper to create legacy conversation (no parent_message_id set)
export function createLegacyConversation(): Message[] {
  return [
    createMessage("msg-1", { role: "user", content: "Hello", created_at: "2025-01-01T00:00:00Z" }),
    createMessage("msg-2", { role: "assistant", content: "Hi!", created_at: "2025-01-01T00:01:00Z" }),
    createMessage("msg-3", { role: "user", content: "Bye", created_at: "2025-01-01T00:02:00Z" }),
  ];
}

/**
 * Tests for messageTree utilities
 *
 * Tests the branching message tree functionality including:
 * - Building message maps for efficient lookups
 * - Computing visible messages based on active leaf
 * - Getting sibling information for version picker
 * - Navigating between message branches
 */

import { describe, it, expect } from "vitest";
import type { Message } from "./api";
import {
  buildMessageMap,
  buildChildrenMap,
  getPathToMessage,
  computeVisibleMessages,
  getMessageSiblings,
  getSiblingInfo,
  hasAlternatives,
  getPreviousSibling,
  getNextSibling,
  findLeafMessages,
  getMessageDepth,
} from "./messageTree";

// Helper to create test messages
function createMessage(
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
function createLinearConversation(): Message[] {
  return [
    createMessage("msg-1", { role: "user", content: "Hello", created_at: "2025-01-01T00:00:00Z" }),
    createMessage("msg-2", { role: "assistant", content: "Hi!", parent_message_id: "msg-1", created_at: "2025-01-01T00:01:00Z" }),
    createMessage("msg-3", { role: "user", content: "How are you?", parent_message_id: "msg-2", created_at: "2025-01-01T00:02:00Z" }),
    createMessage("msg-4", { role: "assistant", content: "I'm good!", parent_message_id: "msg-3", created_at: "2025-01-01T00:03:00Z" }),
  ];
}

// Helper to create a branching conversation with regenerated responses
function createBranchingConversation(): Message[] {
  // User asks question, assistant has 3 alternative responses
  return [
    createMessage("msg-1", { role: "user", content: "Hello" }),
    createMessage("msg-2a", { role: "assistant", content: "Response A", parent_message_id: "msg-1", sibling_index: 0 }),
    createMessage("msg-2b", { role: "assistant", content: "Response B", parent_message_id: "msg-1", sibling_index: 1 }),
    createMessage("msg-2c", { role: "assistant", content: "Response C", parent_message_id: "msg-1", sibling_index: 2 }),
    // User continues from Response B
    createMessage("msg-3", { role: "user", content: "Tell me more", parent_message_id: "msg-2b" }),
    createMessage("msg-4", { role: "assistant", content: "More details...", parent_message_id: "msg-3" }),
  ];
}

// Helper to create legacy conversation (no parent_message_id set)
function createLegacyConversation(): Message[] {
  return [
    createMessage("msg-1", { role: "user", content: "Hello", created_at: "2025-01-01T00:00:00Z" }),
    createMessage("msg-2", { role: "assistant", content: "Hi!", created_at: "2025-01-01T00:01:00Z" }),
    createMessage("msg-3", { role: "user", content: "Bye", created_at: "2025-01-01T00:02:00Z" }),
  ];
}

describe("buildMessageMap", () => {
  it("creates map for O(1) lookups", () => {
    const messages = createLinearConversation();
    const map = buildMessageMap(messages);

    expect(map.size).toBe(4);
    expect(map.get("msg-1")?.content).toBe("Hello");
    expect(map.get("msg-4")?.content).toBe("I'm good!");
    expect(map.get("nonexistent")).toBeUndefined();
  });

  it("handles empty array", () => {
    const map = buildMessageMap([]);
    expect(map.size).toBe(0);
  });
});

describe("buildChildrenMap", () => {
  it("groups children by parent", () => {
    const messages = createBranchingConversation();
    const childrenMap = buildChildrenMap(messages);

    // Root messages (null parent)
    const roots = childrenMap.get(null) ?? [];
    expect(roots).toHaveLength(1);
    expect(roots[0]!.id).toBe("msg-1");

    // Children of msg-1 (the 3 assistant responses)
    const msg1Children = childrenMap.get("msg-1") ?? [];
    expect(msg1Children).toHaveLength(3);
    expect(msg1Children.map(m => m.id)).toEqual(["msg-2a", "msg-2b", "msg-2c"]);
  });

  it("sorts children by sibling_index", () => {
    // Create messages in wrong order
    const messages = [
      createMessage("msg-1"),
      createMessage("msg-2c", { parent_message_id: "msg-1", sibling_index: 2 }),
      createMessage("msg-2a", { parent_message_id: "msg-1", sibling_index: 0 }),
      createMessage("msg-2b", { parent_message_id: "msg-1", sibling_index: 1 }),
    ];

    const childrenMap = buildChildrenMap(messages);
    const children = childrenMap.get("msg-1") ?? [];

    expect(children.map(m => m.sibling_index)).toEqual([0, 1, 2]);
    expect(children.map(m => m.id)).toEqual(["msg-2a", "msg-2b", "msg-2c"]);
  });

  it("handles empty string parent_message_id as null", () => {
    const messages = [
      createMessage("msg-1", { parent_message_id: "" }),
      createMessage("msg-2", { parent_message_id: "msg-1" }),
    ];

    const childrenMap = buildChildrenMap(messages);
    const roots = childrenMap.get(null) ?? [];

    expect(roots).toHaveLength(1);
    expect(roots[0]!.id).toBe("msg-1");
  });
});

describe("getPathToMessage", () => {
  it("returns path from root to target", () => {
    const messages = createLinearConversation();
    const path = getPathToMessage(messages, "msg-4");

    expect(path).toEqual(["msg-1", "msg-2", "msg-3", "msg-4"]);
  });

  it("returns single element for root message", () => {
    const messages = createLinearConversation();
    const path = getPathToMessage(messages, "msg-1");

    expect(path).toEqual(["msg-1"]);
  });

  it("returns empty array for nonexistent message", () => {
    const messages = createLinearConversation();
    const path = getPathToMessage(messages, "nonexistent");

    expect(path).toEqual([]);
  });

  it("handles branching correctly", () => {
    const messages = createBranchingConversation();
    const path = getPathToMessage(messages, "msg-4");

    // Should go through msg-2b branch
    expect(path).toEqual(["msg-1", "msg-2b", "msg-3", "msg-4"]);
  });
});

describe("computeVisibleMessages", () => {
  it("returns messages on path to active leaf", () => {
    const messages = createBranchingConversation();
    const visible = computeVisibleMessages(messages, "msg-4");

    expect(visible.map(m => m.id)).toEqual(["msg-1", "msg-2b", "msg-3", "msg-4"]);
  });

  it("returns different path for different active leaf", () => {
    const messages = createBranchingConversation();
    const visible = computeVisibleMessages(messages, "msg-2a");

    expect(visible.map(m => m.id)).toEqual(["msg-1", "msg-2a"]);
  });

  it("falls back to most recent message when no active leaf", () => {
    const messages = createLinearConversation();
    const visible = computeVisibleMessages(messages);

    // With tree data (parent_message_id), it follows the path to the most recent leaf
    // The function finds the most recent message and traces back to root
    expect(visible.length).toBeGreaterThan(0);
    // Should include the most recent message (msg-4)
    expect(visible[visible.length - 1]!.id).toBe("msg-4");
  });

  it("returns stable empty array for empty input", () => {
    const visible1 = computeVisibleMessages([]);
    const visible2 = computeVisibleMessages([]);

    expect(visible1).toHaveLength(0);
    // Should return same stable reference
    expect(visible1).toBe(visible2);
  });

  it("handles legacy conversations by created_at order", () => {
    const messages = createLegacyConversation();
    const visible = computeVisibleMessages(messages);

    expect(visible.map(m => m.id)).toEqual(["msg-1", "msg-2", "msg-3"]);
  });

  it("preserves original array reference when already sorted", () => {
    const messages = createLegacyConversation();
    const visible = computeVisibleMessages(messages);

    // When messages are already sorted, should return original array
    expect(visible).toBe(messages);
  });

  it("sorts unsorted legacy messages", () => {
    const messages = [
      createMessage("msg-3", { role: "user", created_at: "2025-01-01T00:02:00Z" }),
      createMessage("msg-1", { role: "user", created_at: "2025-01-01T00:00:00Z" }),
      createMessage("msg-2", { role: "assistant", created_at: "2025-01-01T00:01:00Z" }),
    ];

    const visible = computeVisibleMessages(messages);

    expect(visible.map(m => m.id)).toEqual(["msg-1", "msg-2", "msg-3"]);
    // Should be new array since sorting was needed
    expect(visible).not.toBe(messages);
  });
});

describe("getMessageSiblings", () => {
  it("returns all siblings with same parent and role", () => {
    const messages = createBranchingConversation();
    const siblings = getMessageSiblings(messages, "msg-2b");

    expect(siblings).toHaveLength(3);
    expect(siblings.map(m => m.id)).toEqual(["msg-2a", "msg-2b", "msg-2c"]);
  });

  it("returns single element for message with no siblings", () => {
    const messages = createLinearConversation();
    const siblings = getMessageSiblings(messages, "msg-2");

    expect(siblings).toHaveLength(1);
    expect(siblings[0]!.id).toBe("msg-2");
  });

  it("returns empty array for nonexistent message", () => {
    const messages = createLinearConversation();
    const siblings = getMessageSiblings(messages, "nonexistent");

    expect(siblings).toEqual([]);
  });

  it("only includes siblings with same role", () => {
    // Create mixed siblings - user and assistant responses to same parent
    const messages = [
      createMessage("msg-1", { role: "user" }),
      createMessage("msg-2a", { role: "assistant", parent_message_id: "msg-1", sibling_index: 0 }),
      createMessage("msg-2b", { role: "user", parent_message_id: "msg-1", sibling_index: 1 }), // Edited user message
      createMessage("msg-2c", { role: "assistant", parent_message_id: "msg-1", sibling_index: 2 }),
    ];

    const assistantSiblings = getMessageSiblings(messages, "msg-2a");
    expect(assistantSiblings.map(m => m.id)).toEqual(["msg-2a", "msg-2c"]);

    const userSiblings = getMessageSiblings(messages, "msg-2b");
    expect(userSiblings.map(m => m.id)).toEqual(["msg-2b"]);
  });

  it("is sorted by sibling_index", () => {
    const messages = createBranchingConversation();
    const siblings = getMessageSiblings(messages, "msg-2c");

    expect(siblings.map(m => m.sibling_index)).toEqual([0, 1, 2]);
  });
});

describe("getSiblingInfo", () => {
  it("returns current index and total count", () => {
    const messages = createBranchingConversation();
    const info = getSiblingInfo(messages, "msg-2b");

    expect(info.current).toBe(2); // 1-based
    expect(info.total).toBe(3);
    expect(info.siblings).toHaveLength(3);
  });

  it("returns 1/1 for message with no siblings", () => {
    const messages = createLinearConversation();
    const info = getSiblingInfo(messages, "msg-1");

    expect(info.current).toBe(1);
    expect(info.total).toBe(1);
  });

  it("handles first sibling correctly", () => {
    const messages = createBranchingConversation();
    const info = getSiblingInfo(messages, "msg-2a");

    expect(info.current).toBe(1);
  });

  it("handles last sibling correctly", () => {
    const messages = createBranchingConversation();
    const info = getSiblingInfo(messages, "msg-2c");

    expect(info.current).toBe(3);
  });
});

describe("hasAlternatives", () => {
  it("returns true for message with siblings", () => {
    const messages = createBranchingConversation();

    expect(hasAlternatives(messages, "msg-2a")).toBe(true);
    expect(hasAlternatives(messages, "msg-2b")).toBe(true);
    expect(hasAlternatives(messages, "msg-2c")).toBe(true);
  });

  it("returns false for message without siblings", () => {
    const messages = createLinearConversation();

    expect(hasAlternatives(messages, "msg-1")).toBe(false);
    expect(hasAlternatives(messages, "msg-2")).toBe(false);
  });
});

describe("getPreviousSibling", () => {
  it("returns previous sibling", () => {
    const messages = createBranchingConversation();
    const prev = getPreviousSibling(messages, "msg-2b");

    expect(prev?.id).toBe("msg-2a");
  });

  it("returns undefined for first sibling", () => {
    const messages = createBranchingConversation();
    const prev = getPreviousSibling(messages, "msg-2a");

    expect(prev).toBeUndefined();
  });

  it("returns undefined for message with no siblings", () => {
    const messages = createLinearConversation();
    const prev = getPreviousSibling(messages, "msg-1");

    expect(prev).toBeUndefined();
  });
});

describe("getNextSibling", () => {
  it("returns next sibling", () => {
    const messages = createBranchingConversation();
    const next = getNextSibling(messages, "msg-2b");

    expect(next?.id).toBe("msg-2c");
  });

  it("returns undefined for last sibling", () => {
    const messages = createBranchingConversation();
    const next = getNextSibling(messages, "msg-2c");

    expect(next).toBeUndefined();
  });

  it("returns undefined for message with no siblings", () => {
    const messages = createLinearConversation();
    const next = getNextSibling(messages, "msg-1");

    expect(next).toBeUndefined();
  });
});

describe("findLeafMessages", () => {
  it("finds all leaf messages", () => {
    const messages = createBranchingConversation();
    const leaves = findLeafMessages(messages);

    // Leaves are: msg-2a, msg-2c (dead ends), and msg-4 (end of active branch)
    const leafIds = leaves.map(m => m.id).sort();
    expect(leafIds).toEqual(["msg-2a", "msg-2c", "msg-4"]);
  });

  it("returns single leaf for linear conversation", () => {
    const messages = createLinearConversation();
    const leaves = findLeafMessages(messages);

    expect(leaves).toHaveLength(1);
    expect(leaves[0]!.id).toBe("msg-4");
  });

  it("returns all messages as leaves when no parent relationships", () => {
    const messages = createLegacyConversation();
    const leaves = findLeafMessages(messages);

    // All messages are leaves since none are parents
    expect(leaves).toHaveLength(3);
  });
});

describe("getMessageDepth", () => {
  it("returns 0 for root message", () => {
    const messages = createLinearConversation();
    const depth = getMessageDepth(messages, "msg-1");

    expect(depth).toBe(0);
  });

  it("returns correct depth for nested messages", () => {
    const messages = createLinearConversation();

    expect(getMessageDepth(messages, "msg-2")).toBe(1);
    expect(getMessageDepth(messages, "msg-3")).toBe(2);
    expect(getMessageDepth(messages, "msg-4")).toBe(3);
  });

  it("returns -1 for nonexistent message", () => {
    const messages = createLinearConversation();
    const depth = getMessageDepth(messages, "nonexistent");

    expect(depth).toBe(-1);
  });
});

describe("edge cases", () => {
  it("handles single message conversation", () => {
    const messages = [createMessage("msg-1")];

    expect(computeVisibleMessages(messages)).toHaveLength(1);
    expect(getMessageSiblings(messages, "msg-1")).toHaveLength(1);
    expect(findLeafMessages(messages)).toHaveLength(1);
    expect(getMessageDepth(messages, "msg-1")).toBe(0);
  });

  it("handles complex multi-branch tree", () => {
    // Create tree:
    //         msg-1 (user)
    //        /     \
    //     msg-2a  msg-2b (assistant alternatives)
    //       |       |
    //     msg-3a  msg-3b (user follow-ups)
    //       |       |
    //     msg-4a  msg-4b (assistant responses)
    const messages = [
      createMessage("msg-1", { role: "user" }),
      createMessage("msg-2a", { role: "assistant", parent_message_id: "msg-1", sibling_index: 0 }),
      createMessage("msg-2b", { role: "assistant", parent_message_id: "msg-1", sibling_index: 1 }),
      createMessage("msg-3a", { role: "user", parent_message_id: "msg-2a" }),
      createMessage("msg-3b", { role: "user", parent_message_id: "msg-2b" }),
      createMessage("msg-4a", { role: "assistant", parent_message_id: "msg-3a" }),
      createMessage("msg-4b", { role: "assistant", parent_message_id: "msg-3b" }),
    ];

    // Path to msg-4a
    const visibleA = computeVisibleMessages(messages, "msg-4a");
    expect(visibleA.map(m => m.id)).toEqual(["msg-1", "msg-2a", "msg-3a", "msg-4a"]);

    // Path to msg-4b
    const visibleB = computeVisibleMessages(messages, "msg-4b");
    expect(visibleB.map(m => m.id)).toEqual(["msg-1", "msg-2b", "msg-3b", "msg-4b"]);

    // Leaves
    const leaves = findLeafMessages(messages);
    expect(leaves.map(m => m.id).sort()).toEqual(["msg-4a", "msg-4b"]);
  });

  it("handles messages with undefined parent_message_id", () => {
    const messages = [
      createMessage("msg-1", { parent_message_id: undefined }),
      createMessage("msg-2", { parent_message_id: "msg-1" }),
    ];

    const childrenMap = buildChildrenMap(messages);
    const roots = childrenMap.get(null) ?? [];

    expect(roots).toHaveLength(1);
    expect(roots[0]!.id).toBe("msg-1");
  });
});

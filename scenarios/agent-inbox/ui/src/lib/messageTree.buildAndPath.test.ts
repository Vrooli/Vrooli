/**
 * Tests for messageTree utilities - Build maps, path computation, and visible messages
 */

import { describe, it, expect } from "vitest";
import {
  buildMessageMap,
  buildChildrenMap,
  getPathToMessage,
  computeVisibleMessages,
} from "./messageTree";
import {
  createMessage,
  createLinearConversation,
  createBranchingConversation,
  createLegacyConversation,
} from "./messageTree.test.helpers";

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

    const roots = childrenMap.get(null) ?? [];
    expect(roots).toHaveLength(1);
    expect(roots[0]!.id).toBe("msg-1");

    const msg1Children = childrenMap.get("msg-1") ?? [];
    expect(msg1Children).toHaveLength(3);
    expect(msg1Children.map(m => m.id)).toEqual(["msg-2a", "msg-2b", "msg-2c"]);
  });

  it("sorts children by sibling_index", () => {
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

    expect(visible.length).toBeGreaterThan(0);
    expect(visible[visible.length - 1]!.id).toBe("msg-4");
  });

  it("returns stable empty array for empty input", () => {
    const visible1 = computeVisibleMessages([]);
    const visible2 = computeVisibleMessages([]);

    expect(visible1).toHaveLength(0);
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
    expect(visible).not.toBe(messages);
  });
});

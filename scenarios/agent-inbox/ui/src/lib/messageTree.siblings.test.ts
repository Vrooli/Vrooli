/**
 * Tests for messageTree utilities - Sibling info, navigation, leaf finding, depth, edge cases
 */

import { describe, it, expect } from "vitest";
import {
  buildChildrenMap,
  computeVisibleMessages,
  getMessageSiblings,
  getSiblingInfo,
  hasAlternatives,
  getPreviousSibling,
  getNextSibling,
  findLeafMessages,
  getMessageDepth,
} from "./messageTree";
import {
  createMessage,
  createLinearConversation,
  createBranchingConversation,
  createLegacyConversation,
} from "./messageTree.test.helpers";

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
    const messages = [
      createMessage("msg-1", { role: "user" }),
      createMessage("msg-2a", { role: "assistant", parent_message_id: "msg-1", sibling_index: 0 }),
      createMessage("msg-2b", { role: "user", parent_message_id: "msg-1", sibling_index: 1 }),
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

    expect(info.current).toBe(2);
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
    const messages = [
      createMessage("msg-1", { role: "user" }),
      createMessage("msg-2a", { role: "assistant", parent_message_id: "msg-1", sibling_index: 0 }),
      createMessage("msg-2b", { role: "assistant", parent_message_id: "msg-1", sibling_index: 1 }),
      createMessage("msg-3a", { role: "user", parent_message_id: "msg-2a" }),
      createMessage("msg-3b", { role: "user", parent_message_id: "msg-2b" }),
      createMessage("msg-4a", { role: "assistant", parent_message_id: "msg-3a" }),
      createMessage("msg-4b", { role: "assistant", parent_message_id: "msg-3b" }),
    ];

    const visibleA = computeVisibleMessages(messages, "msg-4a");
    expect(visibleA.map(m => m.id)).toEqual(["msg-1", "msg-2a", "msg-3a", "msg-4a"]);

    const visibleB = computeVisibleMessages(messages, "msg-4b");
    expect(visibleB.map(m => m.id)).toEqual(["msg-1", "msg-2b", "msg-3b", "msg-4b"]);

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

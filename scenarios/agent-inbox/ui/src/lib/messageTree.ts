/**
 * Message Tree Utilities
 *
 * Provides utilities for working with branching message trees (ChatGPT-style).
 * Messages form a tree structure where:
 * - parent_message_id points to the parent (null for root messages)
 * - sibling_index orders alternatives (0 = first, 1+ = regenerated)
 * - active_leaf_message_id tracks the current branch
 */

import type { Message } from "./api";

// Stable empty array constant to prevent creating new references on every call.
// CRITICAL: Returning `[]` creates a NEW array each time, which can cause
// infinite re-render loops when used with useMemo dependencies.
const EMPTY_MESSAGES: Message[] = [];

/**
 * Normalize parent_message_id to handle empty strings as null.
 * The backend may return "" for messages without a parent.
 */
function normalizeParentId(parentId: string | undefined | null): string | null {
  if (!parentId || parentId === "") return null;
  return parentId;
}

/**
 * Check if a message has tree data (non-empty parent_message_id).
 */
function hasParent(msg: Message): boolean {
  return !!msg.parent_message_id && msg.parent_message_id !== "";
}

/**
 * Build a map of message ID -> message for O(1) lookups.
 */
export function buildMessageMap(messages: Message[]): Map<string, Message> {
  return new Map(messages.map(m => [m.id, m]));
}

/**
 * Build a map of parent_message_id -> child messages for tree traversal.
 */
export function buildChildrenMap(messages: Message[]): Map<string | null, Message[]> {
  const childrenMap = new Map<string | null, Message[]>();

  for (const msg of messages) {
    const parentId = normalizeParentId(msg.parent_message_id);
    const siblings = childrenMap.get(parentId) ?? [];
    siblings.push(msg);
    childrenMap.set(parentId, siblings);
  }

  // Sort each group by sibling_index
  for (const siblings of childrenMap.values()) {
    siblings.sort((a, b) => a.sibling_index - b.sibling_index);
  }

  return childrenMap;
}

/**
 * Get the path from root to a specific message (inclusive).
 * Returns an array of message IDs from root to the target message.
 */
export function getPathToMessage(messages: Message[], targetId: string): string[] {
  const messageMap = buildMessageMap(messages);
  const path: string[] = [];

  let current = messageMap.get(targetId);
  while (current) {
    path.unshift(current.id);
    const parentId = normalizeParentId(current.parent_message_id);
    current = parentId ? messageMap.get(parentId) : undefined;
  }

  return path;
}

/**
 * Compute the visible messages for a chat based on the active leaf.
 * This follows the path from root to the active leaf, selecting the correct
 * branch at each fork point.
 *
 * For legacy chats (no parent_message_id set), falls back to created_at order.
 */
export function computeVisibleMessages(
  messages: Message[],
  activeLeafId?: string
): Message[] {
  // Use stable EMPTY_MESSAGES constant instead of [] to prevent new reference each call
  if (messages.length === 0) return EMPTY_MESSAGES;

  // Check if this is a legacy linear chat (no branching data)
  // A chat has tree data if any message has a non-empty parent_message_id
  const hasTreeData = messages.some(hasParent);
  if (!hasTreeData) {
    // Legacy behavior: sort by created_at
    // OPTIMIZATION: If messages are already sorted (common case for fresh chats with
    // sequential message creation), return the original array to maintain reference stability.
    // This prevents creating new arrays that could trigger cascading useMemo recalculations.
    const isSorted = messages.every((msg, i) => {
      if (i === 0) return true;
      const prevMsg = messages[i - 1];
      return prevMsg ? new Date(prevMsg.created_at).getTime() <= new Date(msg.created_at).getTime() : true;
    });
    if (isSorted) {
      return messages;
    }
    return [...messages].sort((a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
  }

  // If no active leaf specified, find the most recent message
  if (!activeLeafId) {
    const sorted = [...messages].sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    activeLeafId = sorted[0]?.id;
  }

  if (!activeLeafId) return EMPTY_MESSAGES;

  // Get the path from root to active leaf
  const activePath = new Set(getPathToMessage(messages, activeLeafId));

  // Build children map for traversal
  const childrenMap = buildChildrenMap(messages);
  const result: Message[] = [];

  // Start from root messages (those with no parent)
  const roots = childrenMap.get(null) ?? [];

  // Traverse the tree, following the active path
  function traverse(nodeId: string | null) {
    const children = childrenMap.get(nodeId) ?? [];
    for (const child of children) {
      // Only include messages on the active path
      if (activePath.has(child.id)) {
        result.push(child);
        traverse(child.id);
        break; // Only follow one path at each level
      }
    }
  }

  // Find the root that's on the active path
  for (const root of roots) {
    if (activePath.has(root.id)) {
      result.push(root);
      traverse(root.id);
      break;
    }
  }

  return result;
}

/**
 * Get all sibling messages for a given message (messages with the same parent AND same role).
 * Siblings are alternative versions of the same response, so they must have the same role.
 * Returns an array with just the message itself if there are no siblings.
 */
export function getMessageSiblings(messages: Message[], messageId: string): Message[] {
  const messageMap = buildMessageMap(messages);
  const target = messageMap.get(messageId);

  if (!target) return [];

  const parentId = normalizeParentId(target.parent_message_id);

  // Find all messages with the same parent AND same role
  // Siblings must have the same role (e.g., multiple assistant responses to the same user message)
  return messages
    .filter(m => normalizeParentId(m.parent_message_id) === parentId && m.role === target.role)
    .sort((a, b) => a.sibling_index - b.sibling_index);
}

/**
 * Get sibling information for the version picker display.
 * Returns the current index (1-based) and total count.
 */
export function getSiblingInfo(
  messages: Message[],
  messageId: string
): { current: number; total: number; siblings: Message[] } {
  const siblings = getMessageSiblings(messages, messageId);
  const currentIndex = siblings.findIndex(s => s.id === messageId);

  return {
    current: currentIndex + 1, // 1-based for display
    total: siblings.length,
    siblings,
  };
}

/**
 * Check if a message has alternative versions (siblings).
 */
export function hasAlternatives(messages: Message[], messageId: string): boolean {
  return getMessageSiblings(messages, messageId).length > 1;
}

/**
 * Get the previous sibling for navigation.
 * Returns undefined if already at the first sibling.
 */
export function getPreviousSibling(messages: Message[], messageId: string): Message | undefined {
  const { current, siblings } = getSiblingInfo(messages, messageId);
  if (current <= 1) return undefined;
  return siblings[current - 2]; // -2 because current is 1-based
}

/**
 * Get the next sibling for navigation.
 * Returns undefined if already at the last sibling.
 */
export function getNextSibling(messages: Message[], messageId: string): Message | undefined {
  const { current, total, siblings } = getSiblingInfo(messages, messageId);
  if (current >= total) return undefined;
  return siblings[current]; // current is 1-based, so this gives the next
}

/**
 * Find all leaf messages (messages with no children).
 */
export function findLeafMessages(messages: Message[]): Message[] {
  const parentIds = new Set(messages.map(m => m.parent_message_id).filter(Boolean));
  return messages.filter(m => !parentIds.has(m.id));
}

/**
 * Get the depth of a message in the tree (0 = root).
 */
export function getMessageDepth(messages: Message[], messageId: string): number {
  return getPathToMessage(messages, messageId).length - 1;
}

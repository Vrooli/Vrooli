/**
 * Tests for tool-utils.ts
 *
 * These tests verify skill extraction from tool arguments and document
 * the expected data flow for skill indication in the UI.
 */

import { describe, it, expect } from "vitest";
import { parseToolInput } from "./tool-utils";

describe("parseToolInput", () => {
  describe("skill extraction", () => {
    it("extracts skills from _context_attachments", () => {
      // This is what the ENHANCED arguments look like (from tool_calls table)
      const argsWithSkills = JSON.stringify({
        task: "build a calculator",
        _context_attachments: [
          {
            type: "skill",
            key: "security",
            label: "Security Best Practices",
            content: "Always validate user input...",
            tags: ["security", "best-practices"],
          },
        ],
      });

      const result = parseToolInput(argsWithSkills);

      expect(result.skills).toHaveLength(1);
      expect(result.skills[0]!.key).toBe("security");
      expect(result.skills[0]!.label).toBe("Security Best Practices");
      expect(result.skills[0]!.content).toBe("Always validate user input...");
    });

    it("returns empty skills when no _context_attachments", () => {
      // This is what ORIGINAL arguments look like (from message.tool_calls)
      // BUG: UI was using this instead of the enhanced arguments!
      const argsWithoutSkills = JSON.stringify({
        task: "build a calculator",
      });

      const result = parseToolInput(argsWithoutSkills);

      expect(result.skills).toHaveLength(0);
    });

    it("filters non-skill attachments", () => {
      const argsWithMixedAttachments = JSON.stringify({
        task: "test",
        _context_attachments: [
          { type: "skill", key: "s1", label: "Skill", content: "content" },
          { type: "file", key: "f1", label: "File", content: "data" },
          { type: "skill", key: "s2", label: "Skill 2", content: "content 2" },
        ],
      });

      const result = parseToolInput(argsWithMixedAttachments);

      expect(result.skills).toHaveLength(2);
      expect(result.allAttachments).toHaveLength(3);
    });

    it("handles multiple skills with targeting", () => {
      const argsWithMultipleSkills = JSON.stringify({
        task: "test",
        _context_attachments: [
          { type: "skill", key: "global", label: "Global Skill", content: "applies to all" },
          { type: "skill", key: "targeted", label: "Targeted Skill", content: "only for this tool" },
        ],
      });

      const result = parseToolInput(argsWithMultipleSkills);

      expect(result.skills).toHaveLength(2);
      expect(result.skills[0]!.key).toBe("global");
      expect(result.skills[1]!.key).toBe("targeted");
    });
  });

  describe("argument cleaning", () => {
    it("removes _context_attachments from cleaned arguments", () => {
      const args = JSON.stringify({
        task: "test",
        priority: "high",
        _context_attachments: [
          { type: "skill", key: "s1", label: "Skill", content: "content" },
        ],
      });

      const result = parseToolInput(args);

      // Cleaned JSON should not have _context_attachments
      expect(result.cleanedArgumentsJson).not.toContain("_context_attachments");
      expect(result.cleanedArgumentsJson).toContain("task");
      expect(result.cleanedArgumentsJson).toContain("priority");

      // Arguments object should not have _context_attachments
      expect(result.arguments).toEqual({ task: "test", priority: "high" });
    });

    it("preserves raw arguments for debugging", () => {
      const args = JSON.stringify({
        task: "test",
        _context_attachments: [{ type: "skill", key: "s1", label: "Skill", content: "c" }],
      });

      const result = parseToolInput(args);

      expect(result.rawArguments).toBe(args);
    });
  });

  describe("edge cases", () => {
    it("handles null input", () => {
      const result = parseToolInput(null);
      expect(result.skills).toHaveLength(0);
      expect(result.arguments).toEqual({});
    });

    it("handles undefined input", () => {
      const result = parseToolInput(undefined);
      expect(result.skills).toHaveLength(0);
      expect(result.arguments).toEqual({});
    });

    it("handles empty string input", () => {
      const result = parseToolInput("");
      expect(result.skills).toHaveLength(0);
      expect(result.arguments).toEqual({});
    });

    it("handles invalid JSON gracefully", () => {
      const result = parseToolInput("{invalid json");
      expect(result.skills).toHaveLength(0);
      expect(result.cleanedArgumentsJson).toBe("{invalid json");
    });

    it("handles empty object", () => {
      const result = parseToolInput("{}");
      expect(result.skills).toHaveLength(0);
      expect(result.arguments).toEqual({});
    });

    it("handles primitive value in JSON", () => {
      const result = parseToolInput('"just a string"');
      expect(result.skills).toHaveLength(0);
      expect(result.arguments).toEqual({ value: "just a string" });
    });
  });
});

/**
 * IMPORTANT: Data Flow Documentation
 *
 * There are TWO sources of tool call arguments:
 *
 * 1. message.tool_calls[].function.arguments
 *    - ORIGINAL arguments from AI response
 *    - Does NOT contain _context_attachments (skills)
 *    - Stored in messages table as JSON
 *
 * 2. toolCallRecord.arguments
 *    - ENHANCED arguments with skills injected
 *    - DOES contain _context_attachments when skills are attached
 *    - Stored in tool_calls table
 *
 * BUG: The ToolCallItem component was using source #1 instead of #2,
 * which is why skills were never displayed even when attached.
 *
 * FIX: Use record?.arguments when available, fall back to toolCall.function.arguments
 *
 * Example of the difference:
 *
 * Source #1 (message.tool_calls):
 * {
 *   "task": "build calculator"
 * }
 *
 * Source #2 (toolCallRecord.arguments):
 * {
 *   "task": "build calculator",
 *   "_context_attachments": [
 *     { "type": "skill", "key": "security", "label": "Security", "content": "..." }
 *   ]
 * }
 */

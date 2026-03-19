import { describe, it, expect } from "vitest";
import {
  parseSuggestionsFile,
  buildSuggestionsContent,
  parseSuggestionsObject,
} from "./idea-agent-files";

describe("parseSuggestionsFile", () => {
  it("preserves the notes field through normalization", () => {
    const json = JSON.stringify({
      suggestions: [
        {
          id: "s1",
          suggestion: "Add validation",
          details: "Validates user input",
          status: "accepted",
          notes: "Good idea, implement in v1",
        },
      ],
    });

    const result = parseSuggestionsFile(json);
    expect(result.suggestions).toHaveLength(1);
    expect(result.suggestions[0]?.notes).toBe("Good idea, implement in v1");
  });

  it("normalizes note/response fields to notes", () => {
    const json = JSON.stringify({
      suggestions: [
        { id: "s1", suggestion: "Use caching", note: "from note field" },
        { id: "s2", suggestion: "Add logging", response: "from response field" },
      ],
    });

    const result = parseSuggestionsFile(json);
    expect(result.suggestions[0]?.notes).toBe("from note field");
    expect(result.suggestions[1]?.notes).toBe("from response field");
  });

  it("omits notes when not present", () => {
    const json = JSON.stringify({
      suggestions: [
        { id: "s1", suggestion: "No notes here", status: "pending" },
      ],
    });

    const result = parseSuggestionsFile(json);
    expect(result.suggestions[0]?.notes).toBeUndefined();
  });
});

describe("parseSuggestionsObject", () => {
  it("preserves notes from pre-parsed objects", () => {
    const result = parseSuggestionsObject({
      suggestions: [
        {
          id: "s1",
          suggestion: "Test suggestion",
          status: "rejected",
          notes: "Not applicable to our use case",
        },
      ],
    });
    expect(result.suggestions[0]?.notes).toBe("Not applicable to our use case");
  });
});

describe("buildSuggestionsContent", () => {
  it("round-trips notes through build and parse", () => {
    const original = [
      {
        id: "s1",
        suggestion: "Add feature X",
        status: "accepted" as const,
        notes: "Priority for v1",
      },
      {
        id: "s2",
        suggestion: "Remove feature Y",
        status: "rejected" as const,
        notes: "Still needed for compatibility",
      },
    ];

    const json = buildSuggestionsContent(null, original);
    const parsed = parseSuggestionsFile(json);

    expect(parsed.suggestions).toHaveLength(2);
    expect(parsed.suggestions[0]?.notes).toBe("Priority for v1");
    expect(parsed.suggestions[1]?.notes).toBe("Still needed for compatibility");
  });
});

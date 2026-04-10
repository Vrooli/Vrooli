import { parseCommitGroup, buildContinueMessage } from "./commitGroup";

describe("parseCommitGroup", () => {
  it("parses a standard continuation message", () => {
    expect(parseCommitGroup("web-console TTS p10")).toEqual({
      prefix: "web-console TTS",
      part: 10,
    });
  });

  it("parses multi-word prefixes", () => {
    expect(parseCommitGroup("prompt-manager shared team state p3")).toEqual({
      prefix: "prompt-manager shared team state",
      part: 3,
    });
  });

  it("parses part 1", () => {
    expect(parseCommitGroup("sandboxing reliability p1")).toEqual({
      prefix: "sandboxing reliability",
      part: 1,
    });
  });

  it("parses part 0", () => {
    expect(parseCommitGroup("something p0")).toEqual({
      prefix: "something",
      part: 0,
    });
  });

  it("returns null for non-matching messages", () => {
    expect(parseCommitGroup("fix typo")).toBeNull();
  });

  it("returns null for bare pN with no prefix", () => {
    expect(parseCommitGroup("p5")).toBeNull();
  });

  it("returns null when p has no digits", () => {
    expect(parseCommitGroup("something p")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(parseCommitGroup("")).toBeNull();
  });

  it("handles large part numbers", () => {
    expect(parseCommitGroup("git-control-tower reviews p26")).toEqual({
      prefix: "git-control-tower reviews",
      part: 26,
    });
  });
});

describe("buildContinueMessage", () => {
  it("increments the part number", () => {
    expect(buildContinueMessage("web-console TTS p10")).toBe("web-console TTS p11");
  });

  it("increments across boundaries", () => {
    expect(buildContinueMessage("thing p99")).toBe("thing p100");
  });

  it("increments from 0", () => {
    expect(buildContinueMessage("thing p0")).toBe("thing p1");
  });

  it("returns null for non-matching messages", () => {
    expect(buildContinueMessage("fix typo")).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(buildContinueMessage("")).toBeNull();
  });
});

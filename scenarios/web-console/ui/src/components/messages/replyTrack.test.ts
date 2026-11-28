import { describe, expect, it } from "vitest";
import { replyIndices, replyPosition, seekReply } from "./replyTrack";

// a = reply, u = the operator's own message
const list = (shape: string) => [...shape].map((char) => ({ role: char === "u" ? "user" : "assistant" }));

describe("reply track", () => {
  const track = replyIndices(list("aauaua")); // replies at 0, 1, 3, 5

  it("indexes replies only", () => {
    expect(track).toEqual([0, 1, 3, 5]);
  });

  it("steps over the operator's own messages", () => {
    expect(seekReply(track, 1, 1)).toEqual({ index: 3, moved: 1 });
    expect(seekReply(track, 3, -1)).toEqual({ index: 1, moved: 1 });
  });

  it("covers several replies in one step and reports a short move at an end", () => {
    expect(seekReply(track, 0, 2)).toEqual({ index: 3, moved: 2 });
    expect(seekReply(track, 1, 10)).toEqual({ index: 5, moved: 2 });
    expect(seekReply(track, 3, -10)).toEqual({ index: 0, moved: 2 });
  });

  it("goes nowhere past either end", () => {
    expect(seekReply(track, 5, 1)).toEqual({ index: null, moved: 0 });
    expect(seekReply(track, 0, -1)).toEqual({ index: null, moved: 0 });
    expect(seekReply([], 0, 1)).toEqual({ index: null, moved: 0 });
  });

  it("treats an operator message as sitting between its neighbouring replies", () => {
    expect(seekReply(track, 2, 1)).toEqual({ index: 3, moved: 1 });
    expect(seekReply(track, 2, -1)).toEqual({ index: 1, moved: 1 });
    expect(seekReply(track, 4, 1)).toEqual({ index: 5, moved: 1 });
  });

  it("places the reader among the replies", () => {
    expect(replyPosition(track, 0)).toEqual({ ordinal: 1, total: 4 });
    expect(replyPosition(track, 5)).toEqual({ ordinal: 4, total: 4 });
    expect(replyPosition(track, 2)).toEqual({ ordinal: 2, total: 4 });
  });
});

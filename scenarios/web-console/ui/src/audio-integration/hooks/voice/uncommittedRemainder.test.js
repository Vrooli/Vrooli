/**
 * Phase 1 client red oracle for the streaming-STT backpressure-wedge plan.
 *
 * The single-slot `trailingPartialRef` + `trailingPartialDelta` promotion can
 * only replay the LATEST partial verbatim. It has no notion of how much text
 * the durable segment-finals already committed this turn, so it cannot recover
 * "the full uncommitted remainder BEYOND the last segment-final":
 *
 *   - If the latest partial is the running FULL hypothesis (the natural shape
 *     of a coalesced "latest" partial under the durability contract), promoting
 *     it verbatim DOUBLE-APPENDS the already-committed prefix.
 *   - The correct behaviour is a committed-length cursor: append only the
 *     remainder of the latest partial that lies beyond what segment-finals
 *     already committed.
 *
 * This oracle pins that contract as a pure function `uncommittedRemainder`
 * (the committed-length cursor) that Phase 5 implements in trailingPartial.ts
 * and wires into useVoiceCore's turn-end promotion. It is RED against current
 * code (the function does not exist yet) and GREEN once the committed-length
 * cursor replaces the single overwritten slot.
 */
import { describe, expect, it } from "vitest";
import { uncommittedRemainder } from "./trailingPartial";
describe("uncommittedRemainder (committed-length cursor)", () => {
    it("recovers only the tail beyond the committed segment text (no double-append)", () => {
        // segment-finals committed "hello world"; the last partial the client saw
        // was the running full hypothesis "hello world foo bar". Only " foo bar"
        // is uncommitted and must be promoted — never the whole partial.
        expect(uncommittedRemainder({ committedText: "hello world", latestPartial: "hello world foo bar" })).toBe(" foo bar");
    });
    it("returns a pure current-segment tail unchanged when it does not repeat committed text", () => {
        // kyutai's per-segment partial: the latest partial is already only the
        // uncommitted segment ("foo bar"), with prior segments committed. It is
        // promoted with a leading space to join the committed transcript.
        expect(uncommittedRemainder({ committedText: "hello world", latestPartial: "foo bar" })).toBe(" foo bar");
    });
    it("recovers the full remainder when nothing was committed this turn", () => {
        // The plan's canonical case: partials 'a','a b','a b c', no intervening
        // segment-final, turn ends. The full remainder 'a b c' is recovered.
        expect(uncommittedRemainder({ committedText: "", latestPartial: "a b c" })).toBe("a b c");
    });
    it("promotes nothing when the latest partial adds no new text beyond committed", () => {
        expect(uncommittedRemainder({ committedText: "all committed", latestPartial: "all committed" })).toBeNull();
    });
    it("promotes nothing for an empty or whitespace-only partial", () => {
        expect(uncommittedRemainder({ committedText: "x", latestPartial: "" })).toBeNull();
        expect(uncommittedRemainder({ committedText: "x", latestPartial: "   " })).toBeNull();
    });
    it("does not inject a leading space when the tail opens with closing punctuation", () => {
        expect(uncommittedRemainder({ committedText: "hello world", latestPartial: ", and more" })).toBe(", and more");
    });
});

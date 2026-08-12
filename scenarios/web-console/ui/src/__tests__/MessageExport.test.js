import { describe, it, expect } from "vitest";
import { buildMessageExport, DEFAULT_MESSAGE_EXPORT_FORMAT, estimateTokens, MESSAGE_EXPORT_FORMATS, normalizeExportEvents, } from "../lib/messageExport";
function makeEvent(overrides) {
    return {
        sessionId: "s",
        source: "claude_hook",
        role: "assistant",
        text: `m${overrides.sequence}`,
        speechParagraphs: [],
        summarized: false,
        createdAt: new Date().toISOString(),
        deliveryState: "received",
        ttsState: "idle",
        consumptionState: "seen",
        ...overrides,
    };
}
const ALL_FORMATS = MESSAGE_EXPORT_FORMATS.map((f) => f.id);
describe("MESSAGE_EXPORT_FORMATS", () => {
    it("exposes exactly the four approved formats with Agent XML first", () => {
        expect(ALL_FORMATS).toEqual(["agentXml", "markdown", "quote", "plain"]);
        expect(DEFAULT_MESSAGE_EXPORT_FORMAT).toBe("agentXml");
    });
    it("carries a label and description key for every format", () => {
        for (const descriptor of MESSAGE_EXPORT_FORMATS) {
            expect(descriptor.labelKey).toMatch(/^messageExport\./);
            expect(descriptor.descriptionKey).toMatch(/^messageExport\./);
        }
    });
});
describe("normalizeExportEvents", () => {
    it("sorts by ascending sequence regardless of input order", () => {
        const events = [
            makeEvent({ id: "c", sequence: 30 }),
            makeEvent({ id: "a", sequence: 10 }),
            makeEvent({ id: "b", sequence: 20 }),
        ];
        expect(normalizeExportEvents(events).map((e) => e.id)).toEqual(["a", "b", "c"]);
    });
    it("drops duplicate ids, keeping the first occurrence", () => {
        const events = [
            makeEvent({ id: "a", sequence: 10, text: "first" }),
            makeEvent({ id: "a", sequence: 10, text: "second" }),
            makeEvent({ id: "b", sequence: 20 }),
        ];
        const normalized = normalizeExportEvents(events);
        expect(normalized).toHaveLength(2);
        expect(normalized[0]?.text).toBe("first");
    });
    it("does not mutate the input array", () => {
        const events = [
            makeEvent({ id: "b", sequence: 20 }),
            makeEvent({ id: "a", sequence: 10 }),
        ];
        const snapshot = [...events];
        normalizeExportEvents(events);
        expect(events).toEqual(snapshot);
    });
});
describe("buildMessageExport", () => {
    it.each(ALL_FORMATS)("returns an empty result for an empty selection (%s)", (format) => {
        const result = buildMessageExport([], format);
        expect(result.text).toBe("");
        expect(result.messageCount).toBe(0);
        expect(result.tokenEstimate).toBe(0);
    });
    it.each(ALL_FORMATS)("emits only the selected messages once, in sequence order (%s)", (format) => {
        const events = [
            makeEvent({ id: "b", sequence: 2, role: "assistant", text: "beta" }),
            makeEvent({ id: "a", sequence: 1, role: "user", text: "alpha" }),
            makeEvent({ id: "b", sequence: 2, role: "assistant", text: "beta" }),
        ];
        const result = buildMessageExport(events, format);
        expect(result.messageCount).toBe(2);
        const alphaAt = result.text.indexOf("alpha");
        const betaAt = result.text.indexOf("beta");
        expect(alphaAt).toBeGreaterThanOrEqual(0);
        expect(betaAt).toBeGreaterThan(alphaAt);
        expect(result.text.indexOf("beta", betaAt + 1)).toBe(-1);
    });
    it("renders Agent XML with a neutral conversation root and numbered role messages", () => {
        const events = [
            makeEvent({ id: "a", sequence: 4, role: "user", text: "run the tests" }),
            makeEvent({ id: "b", sequence: 7, role: "assistant", text: "done" }),
        ];
        const result = buildMessageExport(events, "agentXml");
        expect(result.text).toBe("<conversation>\n" +
            '  <message number="4" role="user">\nrun the tests\n  </message>\n' +
            '  <message number="7" role="assistant">\ndone\n  </message>\n' +
            "</conversation>");
    });
    it("does not invent a purpose wrapper, timestamps, or summaries in Agent XML", () => {
        const event = makeEvent({
            id: "a",
            sequence: 1,
            summarized: true,
            speechParagraphs: ["spoken summary"],
            originalSpeechParagraphs: ["original speech"],
            text: "the real text",
        });
        const { text } = buildMessageExport([event], "agentXml");
        expect(text).not.toContain("purpose");
        expect(text).not.toContain("spoken summary");
        expect(text).not.toContain("original speech");
        expect(text).not.toContain(event.createdAt);
        expect(text).toContain("the real text");
    });
    it("escapes special XML characters in message text", () => {
        const event = makeEvent({
            id: "a",
            sequence: 1,
            text: 'if (a < b && c > "d") { emit(<tag/>); }',
        });
        const { text } = buildMessageExport([event], "agentXml");
        expect(text).toContain("if (a &lt; b &amp;&amp; c &gt; &quot;d&quot;) { emit(&lt;tag/&gt;); }");
        expect(text).not.toContain("<tag/>");
    });
    it("renders the Markdown transcript with role/sequence markers and separators", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, role: "user", text: "hello" }),
            makeEvent({ id: "b", sequence: 2, role: "assistant", text: "hi there" }),
        ];
        const { text } = buildMessageExport(events, "markdown");
        expect(text).toBe("**#1 · user**\n\nhello\n\n---\n\n**#2 · assistant**\n\nhi there");
    });
    it("prefixes every line, including empty ones, in quote blocks", () => {
        const event = makeEvent({ id: "a", sequence: 3, role: "user", text: "line one\n\nline two" });
        const { text } = buildMessageExport([event], "quote");
        expect(text).toBe("**#3 · user**\n> line one\n>\n> line two");
    });
    it("renders plain text with stable role and sequence labels", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, role: "user", text: "question" }),
            makeEvent({ id: "b", sequence: 2, role: "assistant", text: "answer" }),
        ];
        const { text } = buildMessageExport(events, "plain");
        expect(text).toBe("[#1] user:\nquestion\n\n[#2] assistant:\nanswer");
    });
    it("keeps multiline content intact in non-quote formats", () => {
        const body = "first line\n\n```ts\nconst x = 1;\n```\nlast line";
        const event = makeEvent({ id: "a", sequence: 1, text: body });
        expect(buildMessageExport([event], "markdown").text).toContain(body);
        expect(buildMessageExport([event], "plain").text).toContain(body);
    });
    it("preserves original roles rather than substituting speech or labels", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, role: "user", text: "u" }),
            makeEvent({ id: "b", sequence: 2, role: "assistant", text: "a" }),
        ];
        for (const format of ALL_FORMATS) {
            const { text } = buildMessageExport(events, format);
            expect(text).toContain("user");
            expect(text).toContain("assistant");
        }
    });
    it("reports the token estimate of the rendered text deterministically", () => {
        const events = [makeEvent({ id: "a", sequence: 1, text: "x".repeat(101) })];
        for (const format of ALL_FORMATS) {
            const result = buildMessageExport(events, format);
            expect(result.tokenEstimate).toBe(estimateTokens(result.text));
            const again = buildMessageExport(events, format);
            expect(again.text).toBe(result.text);
            expect(again.tokenEstimate).toBe(result.tokenEstimate);
        }
    });
});
describe("estimateTokens", () => {
    it("returns zero for empty text", () => {
        expect(estimateTokens("")).toBe(0);
    });
    it("approximates ceil(length / 4)", () => {
        expect(estimateTokens("abcd")).toBe(1);
        expect(estimateTokens("abcde")).toBe(2);
        expect(estimateTokens("x".repeat(400))).toBe(100);
    });
});
// Exhaustiveness guard: a new format added to the union must extend the
// descriptor list, or this assignment stops compiling.
const _allFormatsCovered = ALL_FORMATS;
void _allFormatsCovered;

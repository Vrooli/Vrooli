import type { ConversationEvent } from "../../api/conversation";

function seededRandom(seed: number) {
  let value = seed >>> 0;
  return () => {
    value = (value * 1664525 + 1013904223) >>> 0;
    return value / 0x1_0000_0000;
  };
}

export function makeConversationEvents(count: number, seed = 1): ConversationEvent[] {
  const random = seededRandom(seed);
  return Array.from({ length: count }, (_, index) => {
    const sequence = index + 1;
    const length = Math.min(7000, Math.max(160, Math.round(700 + random() * 600)));
    const prefix = sequence % 10 === 0
      ? `## Event ${sequence}\n\n\`\`\`ts\nconst file = \"/tmp/message-${sequence}.ts\";\n\`\`\`\n\n[open file](/tmp/message-${sequence}.ts)\n\n`
      : `### Event ${sequence}\n\n`;
    const text = (prefix + "reliable message rendering ".repeat(Math.ceil(length / 27))).slice(0, length);
    return {
      id: `fixture-${sequence}`,
      sessionId: "sess-1",
      sequence,
      source: "claude_hook",
      role: sequence % 3 === 0 ? "user" : "assistant",
      text,
      speechParagraphs: [text],
      summarized: false,
      createdAt: new Date(1_700_000_000_000 + sequence * 1000).toISOString(),
      deliveryState: "received",
      ttsState: "idle",
      consumptionState: "seen",
    };
  });
}

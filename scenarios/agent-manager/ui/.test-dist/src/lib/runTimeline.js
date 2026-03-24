export const TIMELINE_CATEGORY_ORDER = [
    "messages",
    "reasoning",
    "tools",
    "errors",
    "compaction",
    "status",
    "logs",
    "artifacts",
    "metrics",
    "redactions",
];
export function createDefaultTimelineFilterState() {
    return {
        mode: "combined",
        categories: {
            messages: true,
            reasoning: true,
            tools: true,
            errors: true,
            status: false,
            logs: false,
            artifacts: false,
            metrics: false,
            compaction: true,
            redactions: false,
        },
    };
}
export function createShowAllTimelineFilterState() {
    return {
        mode: "events",
        categories: TIMELINE_CATEGORY_ORDER.reduce((acc, category) => {
            acc[category] = true;
            return acc;
        }, {}),
    };
}
export function getTimelineModeLabel(mode) {
    switch (mode) {
        case "conversation":
            return "Conversation";
        case "events":
            return "Events";
        default:
            return "Combined";
    }
}
export function getTimelineCategoryLabel(category) {
    switch (category) {
        case "messages":
            return "Messages";
        case "reasoning":
            return "Reasoning";
        case "tools":
            return "Tool Use";
        case "errors":
            return "Errors";
        case "status":
            return "Status";
        case "logs":
            return "Logs";
        case "artifacts":
            return "Artifacts";
        case "metrics":
            return "Metrics";
        case "compaction":
            return "Compaction";
        case "redactions":
            return "Redactions";
    }
}
export function stripImageContextTags(content) {
    return content.replace(/<context\s+[^>]*type="image"[^>]*>\s*<\/context>/g, "").trim();
}
export function isReasoningEvent(event) {
    if (event.data.case !== "log")
        return false;
    const message = String(event.data.value.message ?? "");
    return /^reasoning:\s*/i.test(message) || /^thinking:\s*/i.test(message);
}
export function getTimelineCategory(event) {
    switch (event.data.case) {
        case "message":
            return "messages";
        case "messageDeleted":
            return "redactions";
        case "compaction":
            return "compaction";
        case "toolCall":
        case "toolResult":
            return "tools";
        case "error":
        case "rateLimit":
            return "errors";
        case "status":
        case "progress":
            return "status";
        case "artifact":
            return "artifacts";
        case "metric":
        case "cost":
            return "metrics";
        case "log":
            return isReasoningEvent(event) ? "reasoning" : "logs";
        default:
            return "logs";
    }
}
function toSequenceComparable(event) {
    return typeof event.sequence === "bigint" ? Number(event.sequence) : Number(event.sequence ?? 0);
}
function toTimestampComparable(event) {
    if (!event.timestamp)
        return 0;
    const seconds = Number(event.timestamp.seconds ?? 0);
    const nanos = Number(event.timestamp.nanos ?? 0);
    return (seconds * 1000) + Math.floor(nanos / 1_000_000);
}
export function sortRunEvents(events) {
    return [...events].sort((a, b) => {
        const bySequence = toSequenceComparable(a) - toSequenceComparable(b);
        if (bySequence !== 0)
            return bySequence;
        return toTimestampComparable(a) - toTimestampComparable(b);
    });
}
export function buildTimelineEntries(events) {
    const ordered = sortRunEvents(events);
    const deletedMessageIds = new Set();
    for (const event of ordered) {
        if (event.data.case !== "messageDeleted")
            continue;
        const targetEventId = event.data.value.targetEventId;
        if (targetEventId)
            deletedMessageIds.add(targetEventId);
    }
    const entries = [];
    for (const event of ordered) {
        if (event.data.case === "message") {
            const role = String(event.data.value.role ?? "").toLowerCase();
            if (role !== "user" && role !== "assistant" && role !== "system")
                continue;
            const content = stripImageContextTags(String(event.data.value.content ?? ""));
            const attachments = (event.data.value.attachments ?? []).map((attachment) => ({
                id: attachment.id,
                fileName: attachment.fileName,
                url: attachment.url,
            }));
            if (!content && attachments.length === 0)
                continue;
            entries.push({
                id: event.id,
                kind: "message",
                category: "messages",
                event,
                role,
                content,
                attachments,
                deleted: deletedMessageIds.has(event.id),
            });
            continue;
        }
        const category = getTimelineCategory(event);
        if (category === "messages")
            continue;
        entries.push({
            id: event.id,
            kind: "event",
            category,
            event,
        });
    }
    return entries;
}
export function filterTimelineEntries(entries, filters) {
    return entries.filter((entry) => {
        if (entry.kind === "message") {
            return filters.mode !== "events" && filters.categories.messages;
        }
        if (filters.mode === "conversation")
            return false;
        return filters.categories[entry.category];
    });
}
export function countTimelineEntriesByCategory(entries) {
    const counts = TIMELINE_CATEGORY_ORDER.reduce((acc, category) => {
        acc[category] = 0;
        return acc;
    }, {});
    for (const entry of entries) {
        counts[entry.category] += 1;
    }
    return counts;
}
export function getTimelineEventLabel(entry) {
    switch (entry.category) {
        case "reasoning":
            return "Reasoning";
        case "tools":
            return "Tool";
        case "errors":
            return "Error";
        case "status":
            return "Status";
        case "artifacts":
            return "Artifact";
        case "metrics":
            return "Metric";
        case "compaction":
            return "Compaction";
        case "redactions":
            return "Redaction";
        default:
            return "Log";
    }
}
export function getTimelineEventSummary(entry) {
    const { event } = entry;
    const payload = event.data.value;
    switch (event.data.case) {
        case "log": {
            const message = String(payload.message ?? "Log entry");
            if (entry.category === "reasoning") {
                return message.replace(/^reasoning:\s*/i, "").replace(/^thinking:\s*/i, "");
            }
            return message.replace(/^phase:\s*/i, "");
        }
        case "toolCall":
            return String(payload.toolName ?? "Unknown tool");
        case "toolResult":
            return `${payload.success ? "Completed" : "Failed"} ${String(payload.toolName ?? "tool")}`;
        case "status":
            return `${String(payload.oldStatus ?? "unknown")} -> ${String(payload.newStatus ?? "unknown")}`;
        case "progress":
            return `Progress ${String(payload.percentComplete ?? 0)}%`;
        case "artifact":
            return String(payload.path ?? payload.type ?? "Artifact created");
        case "metric":
            return `${String(payload.name ?? "metric")}: ${String(payload.value ?? 0)}`;
        case "cost":
            return payload.totalCostUsd != null
                ? `Cost update $${Number(payload.totalCostUsd).toFixed(4)}`
                : "Cost update";
        case "error":
            return String(payload.message ?? payload.code ?? "Error");
        case "rateLimit":
            return String(payload.message ?? "Rate limited");
        case "compaction":
            return String(payload.summary ?? payload.trigger ?? "Context compacted");
        case "messageDeleted":
            return `Message ${String(payload.targetEventId ?? "").slice(0, 8)} redacted`;
        default:
            return getTimelineEventLabel(entry);
    }
}
export function buildToolGroupSummary(pairs) {
    const counts = new Map();
    for (const pair of pairs) {
        counts.set(pair.toolName, (counts.get(pair.toolName) ?? 0) + 1);
    }
    const parts = [];
    for (const [name, count] of counts) {
        parts.push(count > 1 ? `${name} ${count}` : name);
    }
    return parts.join(", ");
}
export function groupTimelineEntries(entries) {
    // Index toolResult entries by toolCallId for pairing (when IDs are available)
    const resultsByCallId = new Map();
    for (const entry of entries) {
        if (entry.kind !== "event" || entry.event.data.case !== "toolResult")
            continue;
        const payload = entry.event.data.value;
        const callId = String(payload.toolCallId ?? "");
        if (callId)
            resultsByCallId.set(callId, entry);
    }
    const result = [];
    let toolBuffer = [];
    let activityBuffer = [];
    /** Reasoning entries buffered while waiting to see if more tool calls follow. */
    let pendingReasoning = [];
    const consumedResultIds = new Set();
    function flushBuffer() {
        // First, flush any pending reasoning that wasn't followed by a tool call
        if (toolBuffer.length === 0) {
            for (const r of pendingReasoning)
                result.push(r);
            pendingReasoning = [];
            activityBuffer = [];
            return;
        }
        if (toolBuffer.length >= 2) {
            const first = toolBuffer[0];
            const lastPair = toolBuffer[toolBuffer.length - 1];
            if (!first || !lastPair)
                return;
            const lastEntry = lastPair.result ?? lastPair.call;
            // If there's trailing pending reasoning, include it in the group
            for (const r of pendingReasoning) {
                activityBuffer.push({ kind: "reasoning", entry: r });
            }
            result.push({
                id: `tool-group-${first.call.id}`,
                kind: "tool-group",
                category: "tools",
                pairs: toolBuffer,
                items: activityBuffer,
                summary: buildToolGroupSummary(toolBuffer),
                firstTimestamp: first.call.event.timestamp,
                lastTimestamp: lastEntry.event.timestamp,
            });
        }
        else {
            // Single tool call - emit ungrouped, plus any pending reasoning as standalone
            const pair = toolBuffer[0];
            if (!pair)
                return;
            result.push(pair.call);
            if (pair.result)
                result.push(pair.result);
            for (const r of pendingReasoning)
                result.push(r);
        }
        toolBuffer = [];
        activityBuffer = [];
        pendingReasoning = [];
    }
    for (const entry of entries) {
        if (entry.kind === "event" && entry.event.data.case === "toolCall") {
            // Absorb any pending reasoning into the activity buffer before this tool call
            for (const r of pendingReasoning) {
                activityBuffer.push({ kind: "reasoning", entry: r });
            }
            pendingReasoning = [];
            const payload = entry.event.data.value;
            const toolName = String(payload.toolName ?? "Unknown");
            const callId = String(payload.toolCallId ?? "");
            const matched = callId ? resultsByCallId.get(callId) : undefined;
            if (matched)
                consumedResultIds.add(matched.id);
            const pair = { call: entry, result: matched, toolName };
            toolBuffer.push(pair);
            activityBuffer.push({ kind: "tool-pair", pair });
            continue;
        }
        if (entry.kind === "event" && entry.event.data.case === "toolResult") {
            if (consumedResultIds.has(entry.id))
                continue; // already paired by callId
            // Positional pairing: attach this result to the last unpaired tool call
            let lastUnpaired;
            for (let i = toolBuffer.length - 1; i >= 0; i--) {
                if (!toolBuffer[i]?.result) {
                    lastUnpaired = toolBuffer[i];
                    break;
                }
            }
            if (lastUnpaired) {
                lastUnpaired.result = entry;
                consumedResultIds.add(entry.id);
                continue;
            }
            // Truly orphan result (no pending call at all) - flush buffer, emit standalone
            flushBuffer();
            result.push(entry);
            continue;
        }
        // Reasoning and log events between tool calls get absorbed into the group
        if (entry.kind === "event" &&
            (entry.category === "reasoning" || entry.category === "logs") &&
            toolBuffer.length > 0) {
            pendingReasoning.push(entry);
            continue;
        }
        // Non-tool, non-reasoning entry - flush any accumulated buffer
        flushBuffer();
        result.push(entry);
    }
    flushBuffer();
    return result;
}

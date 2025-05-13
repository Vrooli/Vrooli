import { BotParticipant, Conversation, MessageState } from "./types.js";

/**
 * Responsible for deciding **which bots** should act when a trigger arrives.
 * Could be rule‑based, vector‑similarity, or learned ranking.
 */
export abstract class AgentGraph {
    abstract selectResponders(
        conversation: Conversation,
        trigger: MessageState | { type: "system"; content: string }
    ): Promise<BotParticipant[]>;
}

/**
 * Reads {@link MessageState.respondingBots}. Ignores duplicates / invalid IDs.
 */
export class DirectResponderGraph extends AgentGraph {
    async selectResponders(
        conversation: Conversation,
        trigger: MessageState | { type: "system"; content: string },
    ): Promise<BotParticipant[]> {
        // Only user/assistant messages can carry explicit responders
        if (!("respondingBots" in trigger) || !trigger.respondingBots) return [];

        const allBots = conversation.participants;
        const { respondingBots } = trigger;

        if (respondingBots.includes("@all")) return allBots;

        return allBots.filter((p) => respondingBots.includes(p.id));
    }
}

/* ------------------------------------------------------------------
 * 2) SubscriptionGraph – topic/event based routing ----------------------- */

/** Helpers to match MQTT‑style wildcards.  Replace with a real matcher. */
function matchTopic(pattern: string, topic: string): boolean {
    // TODO: replace with minimatch/mqtt‑pattern implementation
    if (pattern === topic) return true;
    if (pattern.endsWith("/#")) return topic.startsWith(pattern.slice(0, -2));
    return false;
}

/**
 * conversation.meta.eventSubscriptions example:
 * {
 *   "sensor/#":        ["bot_sensor"],
 *   "irrigation/*":    ["bot_irrigator"],
 * }
 */
export class SubscriptionGraph extends AgentGraph {
    async selectResponders(
        conversation: Conversation,
        trigger: MessageState | { type: "system"; content: string },
    ): Promise<BotParticipant[]> {
        const topic =
            "eventTopic" in trigger && trigger.eventTopic ? trigger.eventTopic : undefined;
        if (!topic) return [];

        const subs: Record<string, string[]> =
            (conversation.meta.eventSubscriptions as any) ?? {};

        const responders = new Set<string>();
        for (const [pattern, botIds] of Object.entries(subs)) {
            if (matchTopic(pattern, topic)) botIds.forEach((id) => responders.add(id));
        }
        return conversation.participants.filter((p) => responders.has(p.id));
    }
}

/**
 * **ActiveBotGraph** enforces the OpenAI “swarm” baton pattern.
 *
 * Conversation.meta.activeBotId rules:
 *   • `undefined`   ⇒ Baton is held by the *arbitrator* (first participant
 *      whose meta.role === "arbitrator").
 *   • some bot ID   ⇒ Only that bot may respond until it explicitly hands
 *      the baton off via the special tool‑call `handoff_to_bot`.
 *
 * The baton is a pure *conversation‑level flag*; this graph returns **at most
 * one participant**.  Other graphs later in the chain are skipped whenever a
 * baton holder is set.
 */
export class ActiveBotGraph extends AgentGraph {
    async selectResponders(
        conversation: Conversation,
        _trigger: MessageState | { type: "system"; content: string },
    ): Promise<BotParticipant[]> {
        const active = conversation.meta.activeBotId;
        if (!active) {                         // no baton ⇒ arbitrator
            const arb = conversation.participants.find(p => p.meta?.role === "arbitrator");
            return arb ? [arb] : [];             // should always exist
        }
        const bot = conversation.participants.find(p => p.id === active);
        return bot ? [bot] : [];
    }
}

/* ------------------------------------------------------------------
 * 4) CompositeGraph – combine other graphs ---------------------------------- */

export class CompositeGraph extends AgentGraph {
    constructor(
        private readonly direct = new DirectResponderGraph(),
        private readonly subGraph = new SubscriptionGraph(),
        private readonly activeBotGraph = new ActiveBotGraph(),
    ) { super(); }

    async selectResponders(
        conversation: Conversation,
        trigger: MessageState | { type: "system"; content: string },
    ): Promise<BotParticipant[]> {
        // 1️⃣ explicit responders override everything
        const directRes = await this.direct.selectResponders(conversation, trigger);
        if (directRes.length) return directRes;

        // 2️⃣ topic/event subscriptions
        const subRes = await this.subGraph.selectResponders(conversation, trigger);
        if (subRes.length) return subRes;

        // 3️⃣ fallback arbitrator for **user messages only**
        if ("role" in trigger && trigger.role === "user") {
            return this.activeBotGraph.selectResponders(conversation, trigger);
        }
        return [];
    }
}

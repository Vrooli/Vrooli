import { type BotParticipant, type ChatMessage } from "@vrooli/shared";
const match = require("mqtt-match");
import type { UnifiedSwarmContext } from "../execution/shared/UnifiedSwarmContext.js";
import { type MessageState } from "./types.js";

/**
 * Responsible for deciding **which bots** should act when a trigger arrives.
 * Could be rule‑based, vector‑similarity, or learned ranking.
 */
export abstract class AgentGraph {
    abstract selectResponders(
        context: UnifiedSwarmContext,
        trigger: Pick<ChatMessage, "config">,
    ): Promise<AgentSelectionResult>;
}

export type AgentSelectionResult = {
    /**
     * The bots that should respond to the trigger.
     */
    responders: BotParticipant[];
    /**
     * The strategy used to select the responders.
     */
    strategy: "direct_mention" | "subscription" | "swarm_baton" | "fallback";
};

/**
 * Reads {@link MessageState.respondingBots}. Ignores duplicates / invalid IDs.
 */
export class DirectResponderGraph extends AgentGraph {
    async selectResponders(
        context: UnifiedSwarmContext,
        trigger: Pick<ChatMessage, "config">,
    ): Promise<AgentSelectionResult> {
        const agents = context.execution.agents;
        let respondingBotIds: string[] | undefined;
        const strategy = "direct_mention" as const;

        if (trigger && trigger.config) {
            respondingBotIds = trigger.config.respondingBots;
        }

        if (agents.length === 0 || !respondingBotIds) {
            return { responders: [], strategy };
        }

        if (respondingBotIds.some(id => id === "@all")) {
            return { responders: agents, strategy };
        }

        const matchingAgents = agents.filter((p) => respondingBotIds!.includes(p.id)); // Non-null assertion is safe due to the check above
        const uniqueAgents = Array.from(new Map(matchingAgents.map(bot => [bot.id, bot])).values());
        return { responders: uniqueAgents, strategy };
    }
}

/**
 * conversation.meta.eventSubscriptions example:
 * {
 *   "sensor/#":        ["bot_sensor"],
 *   "irrigation/*":    ["bot_irrigator"],
 * }
 */
export class SubscriptionGraph extends AgentGraph {
    /**
     * Selects bot participants that are subscribed to an event topic present in the trigger message.
     * Subscriptions are defined in `conversation.config.eventSubscriptions`.
     * It uses the {@link matchTopic} function to determine if a bot's subscribed pattern matches
     * the `eventTopic` from the trigger message.
     */
    async selectResponders(
        context: UnifiedSwarmContext,
        trigger: Pick<ChatMessage, "config">,
    ): Promise<AgentSelectionResult> {
        let topic: string | undefined;
        const strategy = "subscription" as const;

        if (trigger && trigger.config) {
            topic = trigger.config.eventTopic;
        }
        if (!topic) {
            return { responders: [], strategy };
        }

        const subs: Record<string, string[]> = context.configuration.eventSubscriptions ?? {};

        const responderIds = new Set<string>();
        for (const [pattern, botIds] of Object.entries(subs)) {
            if (match(pattern, topic)) botIds.forEach((id) => responderIds.add(id));
        }
        const selectedResponders = context.execution.agents.filter((p) => responderIds.has(p.id));
        return { responders: selectedResponders, strategy };
    }
}

/**
 * **ActiveBotGraph** enforces the OpenAI "swarm" baton pattern.
 * See https://github.com/openai/swarm for more details.
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
    private readonly suppressArbitratorWarning: boolean;

    constructor(
        suppressArbitratorWarning = false, // Finding an arbitrator is not strictly required in composite mode
    ) {
        super();
        this.suppressArbitratorWarning = suppressArbitratorWarning;
    }

    async selectResponders(
        context: UnifiedSwarmContext,
        _trigger: Pick<ChatMessage, "config">,
    ): Promise<AgentSelectionResult> {
        const strategy = "swarm_baton" as const;

        const active = context.execution.activeBotId;
        if (!active) {                         // no baton ⇒ arbitrator
            // The correct role is "arbitrator", but since this could be set by a bot, 
            // we'll be more lenient in what's considered an "arbitrator".
            // Ideally this wouldn't be needed, but language models are not always reliable. 
            // So it's best to keep this check.
            // First, specifically look for a participant with the "arbitrator" role.
            let arb = context.execution.agents.find(p => {
                const role = p.meta?.role;
                return typeof role === "string" && role.trim().toLowerCase() === "arbitrator";
            });

            // If no specific "arbitrator" is found, then look for other designated roles.
            if (!arb) {
                const secondaryArbRoles = ["leader", "delegator", "coordinator"]; // Roles other than "arbitrator"
                arb = context.execution.agents.find(p => {
                    const role = p.meta?.role;
                    if (typeof role === "string") {
                        return secondaryArbRoles.includes(role.trim().toLowerCase());
                    }
                    return false;
                });
            }
            if (!arb && !this.suppressArbitratorWarning) {
                // If an arbitrator is expected as per the comment "should always exist"
                // and the logic for finding one, log a warning if none is found,
                // unless warning suppression is active.
                console.warn(`ActiveBotGraph: Expected an arbitrator participant, but none was found. Conversation ID: ${context.swarmId ?? "N/A"}`);
            }
            const responders = arb ? [arb] : [];
            return { responders, strategy };
        }
        const bot = context.execution.agents.find(p => p.id === active);
        const responders = bot ? [bot] : [];
        return { responders, strategy };
    }
}

/**
 * The `CompositeGraph` is responsible for determining which bot(s) should respond to a given trigger
 * by orchestrating a sequence of different selection strategies. Its primary goal is to ensure that
 * a responder is found by deferring to the swarm's self-determined organization and best practices
 * for agent interaction.
 *
 * It follows a specific priority for selecting responders:
 *
 * 1.  **Direct Responders (`DirectResponderGraph`):**
 *     If the trigger message explicitly specifies `respondingBots` in its configuration,
 *     those bots are selected. This allows for direct addressing or commanding of specific bots.
 *
 * 2.  **Subscription-Based Responders (`SubscriptionGraph`):**
 *     If the trigger contains an `eventTopic`, bots that are subscribed to a matching topic
 *     pattern (defined in `conversation.config.eventSubscriptions`) are selected. This enables
 *     event-driven agent activation.
 *
 * 3.  **Active Bot / Arbitrator - Swarm Baton (`ActiveBotGraph`):**
 *     This graph implements the OpenAI "swarm" baton pattern.
 *     - If `conversation.config.activeBotId` is set, only that bot (the "baton holder") is selected.
 *       This bot is expected to manage the conversation flow until it hands off the baton.
 *     - If no `activeBotId` is set, `ActiveBotGraph` attempts to select an "arbitrator" participant
 *       (e.g., one with `meta.role === "arbitrator"`).
 *     The `CompositeGraph` respects this mechanism, allowing the swarm itself to manage
 *     turn-taking and primary responsibility.
 *
 * 4.  **Fallback - First Participant:**
 *     If none of the above strategies yield any responders, and if there are participants
 *     in the conversation, the `CompositeGraph` will select the *first* participant from the
 *     `conversation.participants` array. This acts as a final fallback to ensure that
 *     a trigger is handled if possible, preventing messages from being dropped if other
 *     more specific routing mechanisms don't apply.
 *
 * The overall design ensures that specific routing rules take precedence, followed by the
 * swarm's active agent or arbitrator, and finally, a general fallback, providing a robust
 * way to always find a handler if one exists.
 */
export class CompositeGraph extends AgentGraph {
    constructor(
        private readonly direct = new DirectResponderGraph(),
        private readonly subGraph = new SubscriptionGraph(),
        private readonly activeBotGraph = new ActiveBotGraph(true),
    ) { super(); }

    async selectResponders(
        context: UnifiedSwarmContext,
        trigger: Pick<ChatMessage, "config">,
    ): Promise<AgentSelectionResult> {
        // 1️⃣ explicit responders override everything
        const directResult = await this.direct.selectResponders(context, trigger);
        if (directResult.responders.length) return directResult;

        // 2️⃣ topic/event subscriptions
        const subResult = await this.subGraph.selectResponders(context, trigger);
        if (subResult.responders.length) return subResult;

        // 3️⃣ Swarm baton
        const activeResult = await this.activeBotGraph.selectResponders(context, trigger);
        // If activeResult has responders, OR if an activeBotId was configured in the conversation
        // (meaning the baton was explicitly intended to be active), then this result from ActiveBotGraph
        // is considered definitive for the swarm step. If activeBotId was set but the bot
        // was missing, activeResult.responders will be empty, and we should return that
        // (i.e., no one responds via swarm) rather than proceeding to fallback.
        if (activeResult.responders.length || context.execution?.activeBotId) {
            return activeResult;
        }

        // 4️⃣ If all else fails, just pick the first bot
        if (context.execution.agents.length) {
            return { responders: [context.execution.agents[0]], strategy: "fallback" };
        }
        return { responders: [], strategy: "fallback" };
    }
}

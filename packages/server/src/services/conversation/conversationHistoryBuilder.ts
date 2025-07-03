/* eslint-disable no-magic-numbers */

import type { BotParticipant, MessageConfigObject } from "@vrooli/shared";
import type OpenAI from "openai";
import { DbProvider } from "../../db/provider.js";
import { ChatContextCache, TokenCounter } from "../response/messageStore.js";
import { AIServiceRegistry } from "../response/registry.js";
import type { MessageState } from "./types.js";

/**
 * Represents a single message entry in the per-chat history map.
 */
interface HistoryEntry {
    /** The unique identifier of the message. */
    id: string;
    /** The identifier of the parent message, if any. */
    parentId: string | null;
    /** The identifier of the user who sent this message. */
    userId: string | null;
    /** The raw text content of the message. */
    text: string;
    /** The message configuration (metadata) stored in the database. */
    config: unknown;
    /** The language code of the message. */
    language: string;
    /** ISO timestamp of when the message was created. */
    createdAt: string;
    /** Precomputed token size for this message. */
    tokenSize: number;
}

/**
 * A map of messageId to HistoryEntry for a chat.
 */
type HistoryMap = Record<string, HistoryEntry>;

// Maximum number of history entries to retain per chat.
const MAX_HISTORY_ENTRIES = 1000;

/**
 * The result of building context for a bot turn.
 */
export interface ContextBuildResult {
    /** Ordered full MessageState objects to send as context. */
    messages: MessageState[];
    /** Total tokens counted in the returned history context (excluding world/tools). */
    totalTokens: number;
    /** Whether the context was truncated before reaching the very first message. */
    truncated: boolean;
}

/**
 * Abstract interface for conversation history builders.
 */
export abstract class ConversationHistoryBuilder {
    /**
     * Build a context window for a given chat, starting from an optional message
     * @param chatId - The chat identifier to build context for
     * @param bot - The bot participant generating the response
     * @param aiModel - The model name to budget the context against
     * @param startMessageId - Optional message ID to start context from (null = most recent)
     * @param options - Additional options for the context builder
     */
    abstract build(
        chatId: string,
        bot: BotParticipant,
        aiModel: string,
        startMessageId: string | undefined,
        options: {
            /** OpenAI-format tool schemas to reserve tokens for. */
            tools: OpenAI.Responses.Tool[];
            /** The system message string. */
            systemMessage: string;
        },
    ): Promise<ContextBuildResult>;
}

/**
 * Concrete ConversationHistoryBuilder using a flattened JSON history map in Redis.
 */
export class RedisConversationHistoryBuilder extends ConversationHistoryBuilder {
    /**
     * Builds the conversation context by loading or seeding a flat history map,
     * ensuring token sizes, then walking backwards via parentId until token budget.
     */
    async build(
        chatId: string,
        bot: BotParticipant,
        aiModel: string,
        startMessageId: string | undefined,
        options: { tools: OpenAI.Responses.Tool[]; systemMessage: string },
    ): Promise<ContextBuildResult> {
        const cache = ChatContextCache.get();

        // Prepare the AI service and token counter
        const registry = AIServiceRegistry.get();
        const serviceId = registry.getServiceId(aiModel);
        const service = registry.getService(serviceId);
        // Compute the total model window
        const totalWindow = service.getContextSize(aiModel);
        const tokenCounter = new TokenCounter(service, aiModel);

        // Reserve tokens for the system prompt
        const systemPromptString = options.systemMessage;
        const systemPromptEstimate = service.estimateTokens({ aiModel, text: systemPromptString });
        const systemPromptTokens = systemPromptEstimate.tokens;

        // Reserve tokens for the tool schemas definitions
        const toolsJson = JSON.stringify(options.tools);
        const toolsEstimate = service.estimateTokens({ aiModel, text: toolsJson });
        const toolTokens = toolsEstimate.tokens;

        // Effective budget for history messages
        const budget = totalWindow - (systemPromptTokens + toolTokens);

        // 1️⃣ Load or initialize history map
        let map: HistoryMap | null = await cache.getHistoryMap(chatId);
        if (!map) {
            // Fallback: fetch all messages from the DB
            let recs = await DbProvider.get().chat_message.findMany({
                where: { chatId: BigInt(chatId) },
                include: { parent: { select: { id: true } }, user: { select: { id: true } } },
                orderBy: { createdAt: "asc" },
            });
            // Trim to the most recent entries if over limit
            if (recs.length > MAX_HISTORY_ENTRIES) {
                recs = recs.slice(-MAX_HISTORY_ENTRIES);
            }
            map = {};
            for (const rec of recs) {
                const msg: MessageState = {
                    id: rec.id.toString(),
                    createdAt: rec.createdAt,
                    text: rec.text ?? "",
                    language: rec.language,
                    config: rec.config as any,
                    parent: rec.parent?.id ? { id: rec.parent.id.toString() } : null,
                    user: rec.user?.id ? { id: rec.user.id.toString() } : null,
                };
                const { size } = tokenCounter.ensure(msg, {});
                map[msg.id] = {
                    id: msg.id,
                    parentId: msg.parent?.id ?? null,
                    userId: msg.user?.id ?? null,
                    text: msg.text,
                    config: msg.config,
                    language: msg.language,
                    createdAt: msg.createdAt.toISOString(),
                    tokenSize: size,
                };
            }
            await cache.setHistoryMap(chatId, map);
        }

        // 2️⃣ Ensure token sizes present for any new entries
        let dirty = false;
        for (const entry of Object.values(map!)) {
            if (entry.tokenSize <= 0) {
                const dummy: MessageState = {
                    id: entry.id,
                    createdAt: new Date(entry.createdAt),
                    text: entry.text,
                    language: entry.language,
                    config: entry.config as any,
                    parent: entry.parentId ? { id: entry.parentId } : null,
                    user: entry.userId ? { id: entry.userId } : null,
                };
                const { size } = tokenCounter.ensure(dummy, {});
                entry.tokenSize = size;
                dirty = true;
            }
        }
        if (dirty) {
            await cache.setHistoryMap(chatId, map!);
        }

        // 3️⃣ Walk backwards via parentId under token budget
        const entries = Object.values(map!).sort(
            (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
        );
        const collected: HistoryEntry[] = [];
        let tokens = 0;
        let curId = startMessageId
            ?? (entries.length ? entries[entries.length - 1].id : null);
        while (curId) {
            const e = map![curId];
            if (!e || tokens + e.tokenSize > budget) break;
            collected.push(e);
            tokens += e.tokenSize;
            curId = e.parentId;
        }
        collected.reverse();

        // 4️⃣ Fetch full messages from the database and build MessageState[]
        const collectedIds = collected.map(e => BigInt(e.id));
        const recs = await DbProvider.get().chat_message.findMany({
            where: { id: { in: collectedIds } },
            include: { parent: { select: { id: true } }, user: { select: { id: true } } },
            orderBy: { createdAt: "asc" },
        });
        const messages: MessageState[] = recs.map(rec => ({
            id: rec.id.toString(),
            createdAt: rec.createdAt,
            text: rec.text ?? "",
            language: rec.language,
            config: rec.config as unknown as MessageConfigObject,
            parent: rec.parent?.id ? { id: rec.parent.id.toString() } : null,
            user: rec.user?.id ? { id: rec.user.id.toString() } : null,
        }));

        return {
            messages,
            totalTokens: tokens,
            truncated: curId !== null,
        };
    }
}

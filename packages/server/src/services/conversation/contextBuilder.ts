// /**
//  * Finds bot information for a given bot ID. 
//  * First checks the redis cache, then falls back to the database.
//  */
// export async function getBotInfo(botId: string): Promise<PreMapUserData | null> {
//     let botInfo: PreMapUserData | null = null;

//     // Check Redis cache first
//     await withRedis({
//         process: async (redisClient) => {
//             if (!redisClient) return;
//             const key = ChatContextCache.get().botKey(botId);
//             const rawData = await redisClient.hGetAll(key);
//             if (rawData && Object.keys(rawData).length >= 4) {
//                 // Convert isBot back to boolean
//                 botInfo = {
//                     ...rawData,
//                     isBot: rawData.isBot === "true",
//                 } as unknown as PreMapUserData;
//             }
//         },
//         trace: "0233",
//     });

//     // If not found in cache, query the database
//     if (!botInfo || Object.keys(botInfo).length < 4) { // There are 4 fields in PreMapUserData. Can add better check if needed
//         const botData = await DbProvider.get().user.findUnique({
//             where: { id: BigInt(botId) },
//             select: {
//                 id: true,
//                 name: true,
//                 isBot: true,
//                 botSettings: true,
//             },
//         });
//         botInfo = {
//             ...botData,
//             id: botData?.id.toString() ?? "",
//         } as PreMapUserData;
//         // Store the fetched data in Redis for future use
//         await withRedis({
//             process: async (redisClient) => {
//                 if (!redisClient) return;
//                 // Convert isBot to string, since Redis only accepts string values
//                 const botInfoForRedis = {
//                     ...botInfo,
//                     isBot: botInfo!.isBot.toString(),
//                 };
//                 const key = ChatContextCache.get().botKey(botId);
//                 await redisClient.hSet(key, botInfoForRedis);
//                 await redisClient.expire(key, DAYS_1_S);
//             },
//             trace: "0235",
//         });
//     }

//     return botInfo;
// }

// /**
//  * @returns Valid mentions in a message
//  */
// export function processMentions(
//     messageContent: string,
//     chat: { botParticipants?: string[] },
//     bots: Pick<PreMapUserData, "id" | "name">[],
// ): string[] {
//     // Find markdown links in the message
//     const linkStrings = messageContent.match(/\[([^\]]+)\]\(([^)]+)\)/g);
//     // Get the label and link for each link
//     let links: { label: string, link: string }[] = linkStrings?.map(s => {
//         const [label, link] = s.slice(1, -1).split("](");
//         return { label, link };
//     }) ?? [];

//     // Filter out links where the that aren't a mention. Rules:
//     // 1. Label must start with @
//     // 2. Link must be to this site
//     links = links.filter(l => {
//         if (!l.label.startsWith("@")) return false;
//         try {
//             const url = new URL(l.link);
//             return url.origin === UI_URL;
//         } catch (e) {
//             return false;
//         }
//     });

//     let botsToRespond: string[] = [];
//     // If one of the links is "@Everyone", all bots should respond
//     if (links.some(l => l.label === "@Everyone")) {
//         botsToRespond = chat?.botParticipants ?? [];
//     }
//     // Otherwise, find the bots that were mentioned by name (e.g. "@BotName")
//     else {
//         botsToRespond = links.map(l => {
//             const botId = bots.find(b => b.name === l.label.slice(1))?.id;
//             if (!botId) return null;
//             return botId;
//         }).filter(id => id !== null) as string[];

//         botsToRespond = [...new Set(botsToRespond)];
//     }
//     return botsToRespond;
// }

// /**
//  * Determines which bots should respond based on the message content and chat context.
//  * 
//  * Conditions:
//  * 1. If the message content is blank (likely meaning the message was updated but not its actual content), then no bots should respond.
//  * 2. If the message is not associated with your user ID, no bots should respond.
//  * 3. If there are no bots in the chat, no bots should respond.
//  * 4. If there is one bot in the chat and two participants (i.e., just you and the bot), the bot should respond.
//  * 5. Otherwise, check the message to see if any bots were mentioned.
//  * 
//  * @param message - The message that might trigger bot responses.
//  * @param messageFromUserId - The ID of the user who sent the message.
//  * @param chat - Information about the chat where the message was sent.
//  * @param bots - Information about the bots in the chat.
//  * @param userId - The ID of the user sending the message.
//  * @returns An array of botIds that should respond to the message.
//  */
// export function determineRespondingBots(
//     message: string | null,
//     messageFromUserId: string,
//     chat: { botParticipants?: string[], participantsCount?: number },
//     bots: Pick<PreMapUserData, "id" | "name">[],
//     userId: string,
// ): string[] {
//     if (
//         !chat ||
//         !message ||
//         message.trim() === "" ||
//         !messageFromUserId ||
//         messageFromUserId !== userId ||
//         bots.length === 0
//     ) {
//         return [];
//     }

//     if (bots.length === 1 && chat.participantsCount === 2) {
//         return [bots[0].id];
//     } else {
//         return processMentions(message, chat, bots);
//     }
// }

// /**
//  * Stringifies a list of task context objects. These are used to provide context to the LLM,
//  * typically when referencing a form or other client-side data.
//  * @param taskLabel The label for the current task type, which may be used in the template 
//  * @param taskContexts The list of task context objects to stringify
//  * @param contextTemplateDefault The default template to use for stringifying the data
//  * @returns A stringified list of task context objects
//  */
// export function stringifyTaskContexts(
//     taskLabel: string,
//     taskContexts: TaskContextInfo[],
//     contextTemplateDefault?: string,
// ): string {
//     let result = "";

//     for (let i = 0; i < taskContexts.length; i++) {
//         const { template, templateVariables, data } = taskContexts[i];

//         let contextString = "";
//         const stringifiedData = typeof data === "string" ? data : JSON.stringify(data, null, 2);

//         // If the template is defined, replace template variables with actual data
//         if (template || contextTemplateDefault) {
//             contextString = template || contextTemplateDefault || "";

//             // Define template variables
//             const variables = {
//                 [templateVariables?.task || "<TASK>"]: taskLabel,
//                 [templateVariables?.data || "<DATA>"]: stringifiedData,
//             };
//             // Sort variables from longest to shortest, to avoid issues with overlapping variable names
//             const sortedVariables = Object.entries(variables).sort(
//                 ([varNameA], [varNameB]) => varNameB.length - varNameA.length,
//             );
//             // Replace each variable in the template
//             for (const [varName, value] of sortedVariables) {
//                 // Escape regex special characters in variable names
//                 const escapedVarName = varName.replace(/[-\/\\^$*+?.()|[\]{}]/g, "\\$&");
//                 const regex = new RegExp(escapedVarName, "g");
//                 contextString = contextString.replace(regex, value);
//             }
//         }
//         // Otherwise, default to displaying the data
//         else {
//             contextString = stringifiedData;
//         }

//         result += contextString;

//         // Add spacing between contexts
//         if (i < taskContexts.length - 1) {
//             result += "\n\n";
//         }
//     }

//     return result;
// }


/* -----------------------------------------------------------------------------
 * Conversation context assembly for Vrooli – **patched**
 * -----------------------------------------------------------------------------
 * Fixes applied
 *   1. Snowflake precision: use timestamp‑portion as score for Redis ZSET
 *   2. Key TTL: every cache write sets/reset a 7‑day TTL
 *   3. TokenCounter public getKey() instead of reflective access
 *   4. Atomic deleteMessage via MULTI pipeline
 *   5. Correct `truncated` flag computed inside collect()
 * ---------------------------------------------------------------------------*/

/* eslint-disable no-magic-numbers */
import { RedisClientType } from "redis";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { withRedis } from "../../redisConn.js";
import { LlmServiceRegistry } from "../../tasks/llm/registry.js";
import { type LanguageModelService } from "../../tasks/llm/types.js";
import { JsonSchema } from "./types.js";

/* --------------------------------------------------------------------------
 *  Domain types (copied from shared contracts)                               
 * --------------------------------------------------------------------------*/
export interface BotParticipant {
    id: string;
    name: string;
    meta?: Readonly<Record<string, unknown>> & {
        role?: string;
        extraTools?: JsonSchema[];
        disabledTools?: string[];
        systemPrompt?: string;
    };
}

export interface Conversation {
    id: string;
    participants: BotParticipant[];
    meta: Record<string, unknown> & {
        activeBotId?: string | null;
    };
}

export interface ContextBuildResult {
    messages: Array<{
        role: "system" | "user" | "assistant";
        id: string;
        content: string;
        tokenSize: number;
    }>;
    totalTokens: number;
    truncated: boolean;
}

export abstract class ContextBuilder {
    abstract build(conversation: Conversation, bot: BotParticipant): Promise<ContextBuildResult>;
}

const CACHE_TTL_S = 60 * 60 * 24 * 7; // 7 days

/* --------------------------------------------------------------------------
 *  Redis cache helper                                                        
 * --------------------------------------------------------------------------*/
class ChatContextCache {
    /* — singleton — */
    private constructor() { }
    private static _instance: ChatContextCache | null = null;
    static get(): ChatContextCache {
        return this._instance ?? (this._instance = new ChatContextCache());
    }

    /* Key helpers */
    private messageKey = (id: string) => `msg:${id}`;
    private childrenKey = (id: string) => `children:${id}`;
    private chatKey = (id: string) => `chat:${id}`;

    /* Utility – write + TTL in one helper */
    private async _expire(rc: RedisClientType, ...keys: string[]) {
        const pipe = rc.multi();
        for (const k of keys) pipe.expire(k, CACHE_TTL_S);
        await pipe.exec();
    }

    /* Hash helpers */
    async getMessage(rc: RedisClientType | null, id: string) {
        if (!rc) return null;
        const data = await rc.hGetAll(this.messageKey(id));
        return Object.keys(data).length ? (data as unknown as CachedChatMessage) : null;
    }

    async setMessage(rc: RedisClientType, id: string, data: CachedChatMessage) {
        await rc.hSet(this.messageKey(id), data as any);
        await this._expire(rc, this.messageKey(id));
    }

    async setMessageField(rc: RedisClientType, id: string, field: string, val: string) {
        await rc.hSet(this.messageKey(id), field, val);
        await this._expire(rc, this.messageKey(id));
    }

    /* ZSET helpers */
    /**
     * Use Snowflake timestamp portion (41 bits) as score to stay < 2^53.
     */
    public scoreFromSnowflake(id: string): number {
        return Number((BigInt(id) >> 22n) & 0x1fffffffffffn); // 41‑bit mask
    }

    async addMessageToChat(rc: RedisClientType, chatId: string, messageId: string) {
        const score = this.scoreFromSnowflake(messageId);
        await rc.zAdd(this.chatKey(chatId), { score, value: messageId });
        await this._expire(rc, this.chatKey(chatId));
    }

    async getLatestMessage(rc: RedisClientType | null, chatId: string) {
        if (!rc) return null;
        const res = await rc.zRange(this.chatKey(chatId), -1, -1);
        return res[0] ?? null;
    }

    async getAllMessages(rc: RedisClientType | null, chatId: string) {
        if (!rc) return [] as string[];
        return rc.zRange(this.chatKey(chatId), 0, -1);
    }

    async removeMessageFromChat(rc: RedisClientType, chatId: string, messageId: string) {
        await rc.zRem(this.chatKey(chatId), messageId);
    }

    /* Children helpers */
    async addChild(rc: RedisClientType, parent: string, child: string) {
        await rc.sAdd(this.childrenKey(parent), child);
        await this._expire(rc, this.childrenKey(parent));
    }
    async removeChild(rc: RedisClientType, parent: string, child: string) {
        await rc.sRem(this.childrenKey(parent), child);
    }
    async getChildren(rc: RedisClientType | null, id: string) {
        if (!rc) return [] as string[];
        return rc.sMembers(this.childrenKey(id));
    }

    /* Bulk helpers */
    async deleteKeys(rc: RedisClientType, keys: string[]) {
        if (keys.length) await rc.del(keys);
    }
    async clearChat(rc: RedisClientType, chatId: string) {
        const messageIds = await this.getAllMessages(rc, chatId);
        const keys: string[] = [this.chatKey(chatId)];
        for (const m of messageIds) keys.push(this.messageKey(m), this.childrenKey(m));
        await this.deleteKeys(rc, keys);
    }
}

/* --------------------------------------------------------------------------
 *  Persistent store abstraction                                              
 * --------------------------------------------------------------------------*/
interface MessageRecord {
    id: string;
    text: string | null;
    parentId: string | null;
    userId: string | null;
}

class MessageRepository {
    async getById(id: string): Promise<MessageRecord | null> {
        const rec = await DbProvider.get().chat_message.findUnique({
            where: { id: BigInt(id) },
            select: {
                id: true,
                text: true,
                parent: { select: { id: true } },
                user: { select: { id: true } },
            },
        });
        if (!rec) return null;
        return {
            id,
            text: rec.text,
            parentId: rec.parent?.id?.toString() ?? null,
            userId: rec.user?.id?.toString() ?? null,
        };
    }
}

/* --------------------------------------------------------------------------
 *  Token counting util                                                       
 * --------------------------------------------------------------------------*/
interface StoredTokenCounts { [key: string]: number; }

interface CachedChatMessage {
    id: string;
    tokenCounts: string; // stringified StoredTokenCounts
    parentId?: string;
    userId?: string;
}

class TokenCounter {
    constructor(private svc: LanguageModelService<string>, private model: string) { }
    private _key(): string {
        const { estimationModel, encoding } = this.svc.getEstimationInfo(this.model);
        return `${estimationModel}-${encoding}`;
    }
    /** Public accessor to avoid reflective hacks */
    getKey(): string {
        return this._key();
    }
    parse(raw?: string): StoredTokenCounts {
        if (!raw) return {};
        try {
            return JSON.parse(raw) as StoredTokenCounts;
        } catch {
            return {};
        }
    }
    ensure(text: string, counts: StoredTokenCounts): { counts: StoredTokenCounts; size: number } {
        const k = this._key();
        if (counts[k] === undefined) {
            counts[k] = this.svc.estimateTokens({ aiModel: this.model, text }).tokens;
        }
        return { counts, size: counts[k] };
    }
}

/* --------------------------------------------------------------------------
 *  ConversationAssembler                                                     
 * --------------------------------------------------------------------------*/
interface CollectResult { context: ContextInfo[]; truncated: boolean; totalTokens: number; }

export class ConversationAssembler {
    readonly contextSize: number;
    private cache = ChatContextCache.get();
    private repo = new MessageRepository();
    private token: TokenCounter;

    constructor(private aiModel: string) {
        const svc = LlmServiceRegistry.get().getService(LlmServiceRegistry.get().getServiceId(aiModel));
        this.contextSize = svc.getContextSize(aiModel);
        this.token = new TokenCounter(svc, aiModel);
    }

    /* ---------------- message life‑cycle ops ---------------- */
    async addMessage({ chatId, messageId, text, parentId, userId }: { chatId: string; messageId: string; text: string; parentId?: string | null; userId?: string | null; }) {
        await withRedis({
            trace: "ctx:add", process: async rc => {
                if (!rc) return;
                const { counts } = this.token.ensure(text, {});
                const msg: CachedChatMessage = { id: messageId, tokenCounts: JSON.stringify(counts), parentId: parentId ?? undefined, userId: userId ?? undefined };
                /* pipeline */
                await rc
                    .multi()
                    .hSet(`msg:${messageId}`, msg as any)
                    .zAdd(`chat:${chatId}`, { score: ChatContextCache.get().scoreFromSnowflake(messageId), value: messageId })
                    .exec();
                await rc.expire(`msg:${messageId}`, CACHE_TTL_S);
                await rc.expire(`chat:${chatId}`, CACHE_TTL_S);
                if (parentId) await this.cache.addChild(rc, parentId, messageId);
            }
        });
    }

    async editMessage({ messageId, text, parentId, userId }: { messageId: string; text: string; parentId?: string | null; userId?: string | null; }) {
        await withRedis({
            trace: "ctx:edit", process: async rc => {
                if (!rc) return;
                const existing = await this.cache.getMessage(rc, messageId);
                if (!existing) {
                    logger.error("Message not found for edit", { trace: "ctx:edit:404", messageId });
                    return;
                }
                const { counts } = this.token.ensure(text, {});
                const msg: CachedChatMessage = { ...existing, tokenCounts: JSON.stringify(counts) };
                const pipe = rc.multi();
                if (parentId && parentId !== existing.parentId) {
                    pipe.sRem(`children:${existing.parentId}`, messageId);
                    pipe.sAdd(`children:${parentId}`, messageId);
                    msg.parentId = parentId;
                    pipe.expire(`children:${parentId}`, CACHE_TTL_S);
                    if (existing.parentId) pipe.expire(`children:${existing.parentId}`, CACHE_TTL_S);
                }
                if (userId) msg.userId = userId;
                pipe.hSet(`msg:${messageId}`, msg as any).expire(`msg:${messageId}`, CACHE_TTL_S);
                await pipe.exec();
            }
        });
    }

    async deleteMessage({ chatId, messageId }: { chatId: string; messageId: string }) {
        await withRedis({
            trace: "ctx:del", process: async rc => {
                if (!rc) return;
                const msg = await this.cache.getMessage(rc, messageId);
                if (!msg) return;
                const children = await this.cache.getChildren(rc, messageId);

                const pipe = rc.multi();
                // re‑attach children to parent (or orphan if null)
                for (const child of children) {
                    if (msg.parentId) {
                        pipe.hSet(`msg:${child}`, "parentId", msg.parentId);
                        pipe.sAdd(`children:${msg.parentId}`, child).expire(`children:${msg.parentId}`, CACHE_TTL_S);
                    } else {
                        pipe.hDel(`msg:${child}`, "parentId"); // orphan
                    }
                }
                // remove message keys & chat membership
                pipe.del([`msg:${messageId}`, `children:${messageId}`]).zRem(`chat:${chatId}`, messageId);
                await pipe.exec();
            }
        });
    }

    /* ---------------- context collection ---------------- */
    async collect({ chatId, latestMessage, taskMessage }: { chatId: string; latestMessage?: string | null; taskMessage?: string | null; }): Promise<CollectResult> {
        const context: ContextInfo[] = [];
        let tokensUsed = 0;
        let truncated = false;

        // 1️⃣ Optional task system prompt first
        if (taskMessage) {
            const size = this.token.ensure(taskMessage, {}).size;
            if (size < this.contextSize) {
                context.push({ __type: "text", tokenSize: size, text: taskMessage, userId: null });
                tokensUsed += size;
            }
        }

        // 2️⃣ Walk backwards through thread
        await withRedis({
            trace: "ctx:collect", process: async rc => {
                let cur = latestMessage ?? await this.cache.getLatestMessage(rc, chatId);
                while (cur) {
                    const { details, size } = await this.getMessage(rc, cur);
                    if (!details) break;
                    if (tokensUsed + size > this.contextSize) {
                        truncated = true;
                        break;
                    }
                    context.push({ __type: "message", messageId: cur, tokenSize: size, userId: details.userId ?? null });
                    tokensUsed += size;
                    cur = details.parentId ?? null;
                }
            }
        });

        return { context: context.reverse(), truncated, totalTokens: tokensUsed };
    }

    /* helper fetch */
    private async getMessage(rc: RedisClientType | null, id: string): Promise<{ details: CachedChatMessage | null; size: number }> {
        let cached = await this.cache.getMessage(rc, id);
        const tokenKey = this.token.getKey();
        const parsed = this.token.parse(cached?.tokenCounts ?? "");
        if (parsed[tokenKey] !== undefined) {
            return { details: cached, size: parsed[tokenKey] };
        }
        // cache miss → DB fetch
        const dbRec = await this.repo.getById(id);
        if (!dbRec || !dbRec.text) return { details: null, size: 0 };
        const { counts, size } = this.token.ensure(dbRec.text, {});
        cached = { id, tokenCounts: JSON.stringify(counts), parentId: dbRec.parentId ?? undefined, userId: dbRec.userId ?? undefined };
        if (rc) await this.cache.setMessage(rc, id, cached);
        return { details: cached, size };
    }
}

/* --------------------------------------------------------------------------
 *  ContextBuilder implementation                                             
 * --------------------------------------------------------------------------*/
export class RedisContextBuilder extends ContextBuilder {
    private assembler: ConversationAssembler;
    constructor(private aiModel: string) {
        super();
        this.assembler = new ConversationAssembler(aiModel);
    }

    async build(conversation: Conversation, bot: BotParticipant): Promise<ContextBuildResult> {
        const { context, truncated, totalTokens } = await this.assembler.collect({ chatId: conversation.id });

        const messages = context.map(ci => {
            if (ci.__type === "text") {
                return { role: "system" as const, id: "task", content: ci.text, tokenSize: ci.tokenSize };
            }
            const senderRole = ci.userId === bot.id ? "assistant" as const : "user" as const;
            return { role: senderRole, id: ci.messageId, content: `[[message:${ci.messageId}]]`, tokenSize: ci.tokenSize };
        });

        return { messages, totalTokens, truncated };
    }
}

/* --------------------------------------------------------------------------
 *  ContextInfo union                                                         
 * --------------------------------------------------------------------------*/
interface ContextInfoBase { tokenSize: number; userId: string | null; }
interface MessageCtx extends ContextInfoBase { __type: "message"; messageId: string; }
interface TextCtx extends ContextInfoBase { __type: "text"; text: string; }

type ContextInfo = MessageCtx | TextCtx;

// TODO somewhere before collecting context, should determine tools the bot has available:
// const tools = [
//     ...globalTools,                       // e.g. "handoff_to_bot"
//     ...(bot.meta?.extraTools ?? [])
//   ].filter(t => !bot.meta?.disabledTools?.includes(t.name));
// We should use the toolset to reduce the context size:
// const toolBudget = languageModelService.estimateTokens({
//     aiModel,
//     text: JSON.stringify(tools)   // the same stringify you'll send
//   }).tokens;
//   const maxChatTokens = contextSize - toolBudget - safetyMargin;
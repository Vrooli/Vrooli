/* eslint-disable no-magic-numbers */
import { DAYS_1_S } from "@local/shared";
import { RedisClientType } from "redis";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { withRedis } from "../../redisConn.js";
import { LlmServiceRegistry } from "../../tasks/llm/registry.js";
import { type LanguageModelService } from "../../tasks/llm/types.js";
import { BotParticipant, ConversationState } from "./types.js";

/* --------------------------------------------------------------------------
 *  Domain types (copied from shared contracts)                               
 * --------------------------------------------------------------------------*/
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
    abstract build(conversation: ConversationState, bot: BotParticipant): Promise<ContextBuildResult>;
}

const CACHE_TTL_S = DAYS_1_S * 7;

/* --------------------------------------------------------------------------
 *  Redis cache helper                                                        
 * --------------------------------------------------------------------------*/
class ChatContextCache {
    /* — singleton — */
    // eslint-disable-next-line @typescript-eslint/no-empty-function
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
            },
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
            },
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
            },
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
            },
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

    async build(conversation: ConversationState, bot: BotParticipant): Promise<ContextBuildResult> {
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

import { BotConfig, ChatConfig, LRUCache, type ChatConfigObject, type User } from "@local/shared";
import { type Prisma, type user } from "@prisma/client";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { type BotParticipant, type ConversationState } from "./types.js";
import { WorldModel } from "./worldModel.js";

/* ------------------------------------------------------------------
 * Config & helper types                                              
 * ------------------------------------------------------------------*/

/** Shape that is persisted in chat.config->stats on the chat row. */
export type ConversationStatsPatch = Pick<ChatConfigObject["stats"],
    | "totalToolCalls"
    | "totalCredits">

const DEFAULT_DEBOUNCE_MS = 2_000; // 2s is usually enough to batch bursts
const FINALIZE_TIMEOUT_MS = 10_000; // guard to avoid hanging shutdown
const FINALIZE_POLL_MS = 100;

interface PendingConversationState {
    /** Full state after caller mutation – will be persisted */
    state: ConversationState;
    /** Last version that made it to the DB (null = never) */
    lastStored?: ConversationState | null;
    /** debouncer handle */
    timer?: NodeJS.Timeout | null;
    /** write lock */
    storeInProgress: boolean;
}

type ParticipantDbData = { user: Pick<user, "id" | "name" | "handle" | "isBot" | "botSettings"> };

/* ------------------------------------------------------------------
 * Abstract base                                                      
 * ------------------------------------------------------------------*/

//TODO updating local config isn't ideal. Should first send bus event to invalidate cache, then another event to update cache. Need setup similar to sockets, where we can successfully remove from the correct server(s) cache.
/**
 * ChatStore is a write‑behind cache for long‑lived conversation
 * metadata.  Consumers call `saveState()` whenever they mutate
 * `ChatConfigObject.stats`, etc..  Heavy mutations within a short window are
 * collapsed into a single DB write, dramatically lowering steady‑state
 * write QPS while still guaranteeing eventual consistency.
 */
export abstract class ChatStore {
    private readonly debounceMs: number;
    protected readonly state = new Map<string, PendingConversationState>();

    protected constructor(debounceMs: number = DEFAULT_DEBOUNCE_MS) {
        this.debounceMs = debounceMs;
    }

    /* --------------- ABSTRACT LOW‑LEVEL DB OPS -------------------- */

    protected abstract upsertState(conversationId: string, state: ConversationState): Promise<void>;
    protected abstract fetchState(conversationId: string): Promise<ConversationState>;

    abstract getConversation(id: string): Promise<ConversationState | null>;

    /* ---------------- PUBLIC API (used by ConversationLoop) -------- */

    /**
     * Loads the canonical state for `conversationId` into memory (if not
     * already cached) and returns a **deep clone** that callers may mutate
     * freely without affecting the cache until `saveState` is invoked.
     */
    public async loadState(conversationId: string): Promise<ConversationState> {
        const pending = this.state.get(conversationId);
        if (pending) {
            // return clone so caller cannot mutate in‑cache copy directly
            return structuredClone(pending.state);
        }

        const state = await this.fetchState(conversationId);
        this.state.set(conversationId, {
            state: structuredClone(state),
            lastStored: state,
            storeInProgress: false,
            timer: null,
        });
        return structuredClone(state);
    }

    /**
     * Loads the canonical config for `conversationId` into memory (if not
     * already cached) and returns a **deep clone** that callers may mutate
     * freely without affecting the cache until `saveState` is invoked.
     * @deprecated Use loadState instead
     */
    public async loadConfig(conversationId: string): Promise<ChatConfigObject> {
        const state = await this.loadState(conversationId);
        return state.config;
    }

    /**
     * Mark a **previously cloned & mutated** state as the new source of
     * truth and schedule a debounced persist.
     */
    public saveState(
        conversationId: string,
        updatedState: ConversationState,
    ): void {
        const entry = this.state.get(conversationId);
        if (!entry) {
            // Edge‑case: caller forgot to call loadState first – treat as new
            this.state.set(conversationId, {
                state: structuredClone(updatedState),
                lastStored: null,
                timer: null,
                storeInProgress: false,
            });
            this.schedule(conversationId);
            return;
        }

        // overwrite in‑memory copy
        entry.state = structuredClone(updatedState);
        // debouncer is reset on every mutation burst
        this.schedule(conversationId, entry);
    }

    /** Blocks until all pending writes for **all** conversations flush. */
    public async finalizeSave(timeoutMs = FINALIZE_TIMEOUT_MS): Promise<boolean> {
        const start = Date.now();
        return new Promise((resolve) => {
            const check = () => {
                const pendingWrites = [...this.state.values()].some(
                    (s) => s.storeInProgress || s.timer,
                );
                if (!pendingWrites) return resolve(true);
                if (Date.now() - start > timeoutMs) return resolve(false);
                setTimeout(check, FINALIZE_POLL_MS);
            };
            check();
        });
    }

    /* ---------------------- INTERNALS ----------------------------- */

    private schedule(conversationId: string, entry?: PendingConversationState) {
        const stateEntry = entry ?? this.state.get(conversationId);
        if (!stateEntry) return; // Skip if no entry exists

        if (stateEntry.timer) clearTimeout(stateEntry.timer);
        stateEntry.timer = setTimeout(() => this.flush(conversationId).catch(console.error), this.debounceMs);
    }

    /** Flushes one conversation's pending state (called by debounce). */
    private async flush(conversationId: string): Promise<void> {
        const rec = this.state.get(conversationId);
        if (!rec || rec.storeInProgress) return; // already flushing or removed

        rec.timer = null;
        rec.storeInProgress = true;
        try {
            await this.upsertState(conversationId, rec.state);
            rec.lastStored = structuredClone(rec.state);
        } catch (err) {
            console.error("ChatPersistence flush failed", conversationId, err);
        } finally {
            rec.storeInProgress = false;
            // If callers mutated again while we were flushing, ensure another flush
            if (rec.timer === null && rec.lastStored !== rec.state) {
                this.schedule(conversationId, rec);
            }
        }
    }
}

/* ------------------------------------------------------------------
 * Concrete Prisma implementation                                     
 * ------------------------------------------------------------------*/


const chatParticipantSelect = {
    id: true,
    user: {
        select: {
            id: true,
            handle: true,
            isBot: true,
            name: true,
            botSettings: true,
        },
    },
} satisfies Prisma.chat_participantsSelect;

function mapParticipant(participant: ParticipantDbData): BotParticipant {
    const { id, name } = participant.user;
    return {
        id: id.toString(),
        config: BotConfig.parse(participant.user as Pick<User, "botSettings">, logger),
        name,
        meta: {}, //TODO need to get this somewhere else, since it contains information calculated by the current conversation/swarm/routine, rather than information tied to the bot object itself and stored in the database
    };
}



function diffPatch<T extends object>(from: T, to: T): Partial<T> | null {
    let changed = false;
    // Create appropriate initial value based on input type
    const patch = Array.isArray(to)
        ? [] as unknown as Partial<T>
        : {} as Partial<T>;

    for (const [k, v] of Object.entries(to)) {
        const prev = (from as Record<string, unknown>)[k];
        if (typeof v === "object" && v !== null && !Array.isArray(v)) {
            const child = diffPatch(prev ?? {}, v);
            if (child) {
                patch[k] = child;
                changed = true;
            }
        } else if (v !== prev) {
            patch[k] = v;
            changed = true;
        }
    }
    return changed ? patch : null;
}


export class PrismaChatStore extends ChatStore {
    constructor() {
        super();
    }

    protected async fetchState(conversationId: string): Promise<ConversationState> {
        return await this.getConversation(conversationId) || this.createEmptyState(conversationId);
    }

    private createEmptyState(conversationId: string): ConversationState {
        return {
            id: conversationId,
            config: ChatConfig.default().export(),
            participants: [],
            availableTools: [],
            worldModel: new WorldModel(),
        };
    }

    protected async upsertState(id: string, state: ConversationState): Promise<void> {
        const entry = this.state.get(id);
        const last = entry?.lastStored ?? this.createEmptyState(id);

        // For config, we use diffPatch to efficiently update only changed parts
        const configPatch = diffPatch(last.config, state.config);

        await DbProvider.get().chat.update({
            where: { id: BigInt(id) },
            data: {
                // Update config if changed
                ...(configPatch && {
                    config: {
                        path: [],
                        value: configPatch as Prisma.InputJsonValue,
                    },
                }),
            },
        });
    }

    /** 
     * Fetches a chat and its related data to construct the ConversationState.
     * This is the primary method for loading the full state from the database.
    */
    async getConversation(id: string): Promise<ConversationState | null> {
        const chat = await DbProvider.get().chat.findUnique({
            where: { id: BigInt(id) },
            select: {
                id: true,
                participants: { select: chatParticipantSelect },
                config: true,
            },
        });

        if (!chat) return null;

        const config = (chat.config as ChatConfigObject | null) ?? ChatConfig.default().export();
        const participants = chat.participants.map(mapParticipant);

        // Rehydrate WorldModel from persisted config if available
        let worldModelInstance: WorldModel;
        if (config.worldModelConfig) {
            worldModelInstance = new WorldModel({
                systemMessage: config.worldModelConfig.systemMessage,
                goal: config.worldModelConfig.goal,
                subTasks: config.worldModelConfig.subTasks || [],
                sharedScratchpad: config.worldModelConfig.scratchpad,
                resources: config.worldModelConfig.resources || [],
                routineCalls: config.worldModelConfig.routineCalls || [],
            });
        } else {
            worldModelInstance = new WorldModel(); // Default if no persisted config
        }

        return {
            id: chat.id.toString(),
            config,
            participants,
            availableTools: [], // TODO: Populate this from chat config or elsewhere
            worldModel: worldModelInstance,
        };
    }
}

/**
 * Abstraction over both persistent config and in-memory runtime state.
 */
export interface ConversationStateStore {
    /**
     * Load (or reload) the full conversation state.
     * @param invalidate force a fresh load from the persistent store
     */
    get(conversationId: string, invalidate?: boolean): Promise<ConversationState | null>;

    /**
     * Persist an updated config (debounced internally by ChatStore).
     */
    updateConfig(conversationId: string, config: ChatConfigObject): void;

    /**
     * Remove the state from the in-memory cache (LRU eviction or manual).
     */
    del(conversationId: string): void;
}

const DEFAULT_MAX_CONCURRENT_CONVERSATIONS = 1_000;

/**
 * Implements ConversationStateStore by wrapping a ChatStore and an LRU cache.
 * 
 * This allows us to use the database for storing long-term conversation state,
 * while keeping a small number of conversations in memory for quick access 
 * and additional short-term state.
 */
export class CachedConversationStateStore implements ConversationStateStore {
    private readonly cache: LRUCache<string, ConversationState>;

    constructor(
        private readonly chatStore: ChatStore,
        maxEntries = DEFAULT_MAX_CONCURRENT_CONVERSATIONS,
    ) {
        this.cache = new LRUCache<string, ConversationState>({ limit: maxEntries });
    }

    public async get(
        conversationId: string,
        invalidate = false,
    ): Promise<ConversationState | null> {
        // If forced reload or not cached, fetch fresh
        if (invalidate || !this.cache.has(conversationId)) {
            const fresh = await this.chatStore.getConversation(conversationId);
            if (!fresh) return null;
            this.cache.set(conversationId, fresh);
        }
        // Guaranteed to exist after above
        return this.cache.get(conversationId) ?? null;
    }

    public updateConfig(
        conversationId: string,
        config: ChatConfigObject,
    ): void {
        // Update in-memory cache
        const existingState = this.cache.get(conversationId);
        const result = {
            // Fallback and existing state
            participants: [],
            ...existingState,
            id: conversationId,
            // Updated config
            config,
            // Preserve or default new fields
            availableTools: existingState?.availableTools ?? [],
            worldModel: existingState?.worldModel ?? new WorldModel(),
        } satisfies ConversationState;
        this.cache.set(conversationId, result);
        // Update the state in the database
        this.chatStore.saveState(conversationId, result);
    }

    public del(conversationId: string): void {
        this.cache.delete(conversationId);
    }
}

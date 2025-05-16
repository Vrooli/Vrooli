import { ChatConfig, ChatConfigObject, LRUCache } from "@local/shared";
import { Prisma, user } from "@prisma/client";
import { DbProvider } from "../../db/provider.js";
import { BotParticipant, ConversationState } from "./types.js";

/* ------------------------------------------------------------------
 * Config & helper types                                              
 * ------------------------------------------------------------------*/

/** Shape that is persisted in chat.config->stats on the chat row. */
export type ConversationStatsPatch = Pick<ChatConfigObject["stats"],
    | "totalToolCalls"
    | "totalCredits"
    | "lastTurnEndedAt">

const DEFAULT_DEBOUNCE_MS = 2_000; // 2s is usually enough to batch bursts
const FINALIZE_TIMEOUT_MS = 10_000; // guard to avoid hanging shutdown
const FINALIZE_POLL_MS = 100;

interface PendingConversationState {
    /** Full config after caller mutation – will be persisted */
    config: ChatConfigObject;
    /** Last version that made it to the DB (null = never) */
    lastStored?: ChatConfigObject | null;
    /** debouncer handle */
    timer?: NodeJS.Timeout | null;
    /** write lock */
    storeInProgress: boolean;
}

type ParticipantDbData = { user: Pick<user, "id" | "name" | "handle" | "isBot"> };

/* ------------------------------------------------------------------
 * Abstract base                                                      
 * ------------------------------------------------------------------*/

/**
 * ChatStore is a write‑behind cache for long‑lived conversation
 * metadata.  Consumers call `saveState()` whenever they mutate
 * `ChatConfigObject.stats`.  Heavy mutations within a short window are
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

    protected abstract upsertConfig(conversationId: string, config: ChatConfigObject): Promise<void>;
    protected abstract fetchConfig(conversationId: string): Promise<ChatConfigObject>;

    abstract getConversation(id: string): Promise<ConversationState | null>;

    /* ---------------- PUBLIC API (used by ConversationLoop) -------- */

    /**
     * Loads the canonical config for `conversationId` into memory (if not
     * already cached) and returns a **deep clone** that callers may mutate
     * freely without affecting the cache until `saveState` is invoked.
     */
    public async loadConfig(conversationId: string): Promise<ChatConfigObject> {
        const pending = this.state.get(conversationId);
        if (pending) {
            // return clone so caller cannot mutate in‑cache copy directly
            return structuredClone(pending.config);
        }

        const cfg = await this.fetchConfig(conversationId);
        this.state.set(conversationId, {
            config: structuredClone(cfg),
            lastStored: cfg,
            storeInProgress: false,
            timer: null,
        });
        return structuredClone(cfg);
    }

    /**
     * Mark a **previously cloned & mutated** config as the new source of
     * truth and schedule a debounced persist.
     */
    public saveState(
        conversationId: string,
        updatedConfig: ChatConfigObject,
    ): void {
        const entry = this.state.get(conversationId);
        if (!entry) {
            // Edge‑case: caller forgot to call loadConfig first – treat as new
            this.state.set(conversationId, {
                config: structuredClone(updatedConfig),
                lastStored: null,
                timer: null,
                storeInProgress: false,
            });
            this.schedule(conversationId);
            return;
        }

        // overwrite in‑memory copy
        entry.config = structuredClone(updatedConfig);
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

    /** Flushes one conversation's pending config (called by debounce). */
    private async flush(conversationId: string): Promise<void> {
        const rec = this.state.get(conversationId);
        if (!rec || rec.storeInProgress) return; // already flushing or removed

        rec.timer = null;
        rec.storeInProgress = true;
        try {
            await this.upsertConfig(conversationId, rec.config);
            rec.lastStored = structuredClone(rec.config);
        } catch (err) {
            console.error("ChatPersistence flush failed", conversationId, err);
        } finally {
            rec.storeInProgress = false;
            // If callers mutated again while we were flushing, ensure another flush
            if (rec.timer === null && rec.lastStored !== rec.config) {
                this.schedule(conversationId, rec);
            }
        }
    }

    public initializeTurnState(): Pick<ConversationState, "queue" | "turnStats" | "turnCounter"> {
        return {
            queue: new Map(),
            turnStats: { toolCalls: 0, botToolCalls: 0, creditsUsed: 0n },
            turnCounter: 0,
        };
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
        },
    },
} satisfies Prisma.chat_participantsSelect;

function mapParticipant(participant: ParticipantDbData): BotParticipant {
    return {
        id: participant.user.id.toString(),
        name: participant.user.name,
        meta: {}, //TODO need to get this somewhere. Isn't stored in the user because it's chat-specific
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

    protected async fetchConfig(conversationId: string): Promise<ChatConfigObject> {
        const row = await DbProvider.get().chat.findUnique({
            where: { id: BigInt(conversationId) },
            select: { config: true },
        });
        if (!row?.config) return {} as ChatConfigObject; // fallback
        return row.config as unknown as ChatConfigObject;
    }

    protected async upsertConfig(id: string, cfg: ChatConfigObject): Promise<void> {
        const entry = this.state.get(id);
        const last = entry?.lastStored ?? {};
        const patch = diffPatch(last, cfg);
        if (!patch) return;

        /* Prisma's JSON path update merges objects when path === [] */
        await DbProvider.get().chat.update({
            where: { id: BigInt(id) },
            data: {
                config: {
                    path: [],
                    value: patch as Prisma.InputJsonValue,
                },
            },
        });
    }

    async getConversation(id: string): Promise<ConversationState | null> {
        const row = await DbProvider.get().chat.findUnique({
            where: { id: BigInt(id) },
            select: {
                id: true,
                participants: { select: chatParticipantSelect },
                config: true,
                turnCounter: true,
            },
        });
        if (!row) return null;
        const state: ConversationState = {
            // Initialize runtime state
            ...this.initializeTurnState(),
            // Grab persisted data
            id: row.id.toString(),
            config: (row.config ?? {}) as unknown as ChatConfigObject,
            participants: row.participants.map(mapParticipant),
            turnCounter: row.turnCounter,
        };
        return state;
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
     * Persist an updated turn counter.
     */
    updateTurnCounter(conversationId: string, turnCounter: number): void;

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
        this.cache = new LRUCache<string, ConversationState>(maxEntries);
    }

    public async get(
        conversationId: string,
        invalidate = false,
    ): Promise<ConversationState | null> {
        // If forced reload or not cached, fetch fresh
        if (invalidate || this.cache.get(conversationId) === undefined) {
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
            ...this.chatStore.initializeTurnState(),
            ...existingState,
            id: conversationId,
            // Updated config
            config,
        } satisfies ConversationState;
        this.cache.set(conversationId, result);
        // Delegate persistence (debounced/diff-patched) to ChatStore
        this.chatStore.saveState(conversationId, config);
    }

    public updateTurnCounter(
        conversationId: string,
        turnCounter: number,
    ): void {
        // Update in-memory cache
        const existingState = this.cache.get(conversationId);
        const result = {
            // Fallback and existing state
            config: ChatConfig.default().export(),
            participants: [],
            ...this.chatStore.initializeTurnState(),
            ...existingState,
            id: conversationId,
            // Updated turn counter
            turnCounter,
        } satisfies ConversationState;
        this.cache.set(conversationId, result);
    }

    public del(conversationId: string): void {
        this.cache.delete(conversationId);
    }
}

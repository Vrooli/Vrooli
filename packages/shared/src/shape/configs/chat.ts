// Import common types and utilities
import { API_CREDITS_MULTIPLIER } from "../../consts/api.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { DAYS_1_MS, MINUTES_10_MS, MINUTES_1_MS } from "../../consts/numbers.js";
import { BaseConfig, type BaseConfigObject } from "./baseConfig.js";
import { type StringifyMode } from "./utils.js";

// Keep track of config version for future compatibility
const LATEST_CONFIG_VERSION = "1.0";

/**
 * Represents all data that can be stored in a chat's stringified config.
 * Corresponds to Conversation.meta in the server-side types.
 */
export interface ChatConfigObject extends BaseConfigObject {
    /** The ID of the bot that is currently active in the conversation. */
    activeBotId?: string;
    /** Simple event subscriptions map to route events to bots. */
    eventSubscriptions?: Record<string, string[]>;
    /** Limits for the conversation. */
    limits?: {
        /** Maximum number of events per turn */
        maxEventsPerTurn?: number;
        /** Maximum number of tool calls per bot turn */
        maxToolCallsPerBotTurn?: number;
        /** Maximum number of tool calls per turn */
        maxToolCallsPerTurn?: number;
        /** Maximum number of tool calls per conversation */
        maxToolCalls?: number;
        /** Maximum API credits per turn, as a stringified bigint */
        maxCreditsPerTurn?: string;
        /** Maximum API credits per conversation, as a stringified bigint */
        maxCredits?: string;
        /** Maximum duration per turn (ms) */
        maxDurationTurnMs?: number;
        /** Maximum duration per conversation (ms) */
        maxDurationMs?: number;
        /** Delay between turns (ms) */
        turnDelayMs?: number;
    };
    /** Stats for the conversation. */
    stats: {
        /** since convo began */
        totalToolCalls: number;
        totalCredits: string; // Stringified bigint
        /** conversation creation or first‑seen time */
        startedAt: number | null;
        /** wall‑clock of last completed turn */
        lastTurnEndedAt: number | null;
    };
}

/**
 * Top-level Chat config that encapsulates all chat-related configuration data.
 */
export class ChatConfig extends BaseConfig<ChatConfigObject> {
    /** Current active bot ID in the conversation */
    activeBotId?: ChatConfigObject["activeBotId"];
    /** Configured conversation limits */
    limits?: ChatConfigObject["limits"];
    /** Stats for the conversation */
    stats: ChatConfigObject["stats"];

    constructor({ config }: { config: ChatConfigObject }) {
        super(config);
        this.activeBotId = config.activeBotId;
        this.limits = config.limits;
        this.stats = config.stats;
    }

    /**
     * Parse and instantiate a ChatConfig from a serialized string.
     * Populates defaults when useFallbacks is true.
     */
    static parse(
        version: { config: ChatConfigObject },
        logger: PassableLogger,
        opts?: { mode?: StringifyMode; useFallbacks?: boolean },
    ): ChatConfig {
        return super.parseBase<ChatConfigObject, ChatConfig>(
            version.config,
            logger,
            (cfg) => {
                if (opts?.useFallbacks ?? true) {
                    cfg.limits ??= ChatConfig.defaultLimits();
                    cfg.stats ??= ChatConfig.defaultStats();
                }
                return new ChatConfig({ config: cfg });
            },
            { mode: opts?.mode },
        );
    }

    /**
     * Creates a default ChatConfig with no active bot and default limits.
     */
    static default(): ChatConfig {
        const config: ChatConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
            limits: ChatConfig.defaultLimits(),
            stats: ChatConfig.defaultStats(),
        };
        return new ChatConfig({ config });
    }

    /**
     * Exports the config to a plain object.
     */
    override export(): ChatConfigObject {
        return {
            ...super.export(),
            activeBotId: this.activeBotId,
            limits: this.limits,
            stats: this.stats,
        };
    }

    /**
     * Provides default conversation limits (empty object of optional fields).
     */
    static defaultLimits(): ChatConfigObject["limits"] {
        return {};
    }

    /**
     * Default stats for a conversation.
     */
    static defaultStats(): ChatConfigObject["stats"] {
        return {
            totalToolCalls: 0,
            totalCredits: "0",
            startedAt: new Date().getTime(),
            lastTurnEndedAt: null,
        };
    }

    /**
     * Absolute caps for conversation limits.
     */
    public static ABSOLUTE_LIMITS = {
        maxEventsPerTurn: 100,
        maxToolCallsPerBotTurn: 50,
        maxToolCallsPerTurn: 100,
        maxToolCalls: 1000,
        maxCreditsPerTurn: API_CREDITS_MULTIPLIER * BigInt(2),
        maxCredits: API_CREDITS_MULTIPLIER * BigInt(100),
        maxDurationTurnMs: MINUTES_10_MS,
        maxDurationMs: DAYS_1_MS,
        turnDelayMs: MINUTES_1_MS,
    } as const;

    /**
     * Sets or replaces the active bot ID.
     */
    setActiveBotId(activeBotId: ChatConfigObject["activeBotId"]): void {
        this.activeBotId = activeBotId;
    }

    /**
     * Sets or replaces the conversation limits.
     */
    setLimits(limits: ChatConfigObject["limits"]): void {
        this.limits = limits;
    }

    /**
     * Merges configured limits with absolute caps to produce effective limits.
     */
    public getEffectiveLimits(): Required<NonNullable<ChatConfigObject["limits"]>> {
        const defined = this.limits ?? {};
        const { maxToolCallsPerBotTurn, maxToolCallsPerTurn, maxToolCalls, maxCreditsPerTurn, maxCredits, maxDurationTurnMs, maxDurationMs, turnDelayMs, maxEventsPerTurn } = ChatConfig.ABSOLUTE_LIMITS;
        function inNumberRange(max: number, val?: number, min = 0): number {
            return Math.max(min, Math.min(max, val ?? max));
        }
        function inBigIntRange(max: bigint, val?: string, min = BigInt(0)): bigint {
            const definedVal = val !== undefined ? BigInt(val) : undefined;
            return BigInt(inNumberRange(Number(max), definedVal !== undefined ? Number(definedVal) : undefined, Number(min)));
        }
        return {
            maxEventsPerTurn: inNumberRange(maxEventsPerTurn, defined.maxEventsPerTurn),
            maxToolCallsPerBotTurn: inNumberRange(maxToolCallsPerBotTurn, defined.maxToolCallsPerBotTurn),
            maxToolCallsPerTurn: inNumberRange(maxToolCallsPerTurn, defined.maxToolCallsPerTurn),
            maxToolCalls: inNumberRange(maxToolCalls, defined.maxToolCalls),
            maxCreditsPerTurn: inBigIntRange(maxCreditsPerTurn, defined.maxCreditsPerTurn).toString(),
            maxCredits: inBigIntRange(maxCredits, defined.maxCredits).toString(),
            maxDurationTurnMs: inNumberRange(maxDurationTurnMs, defined.maxDurationTurnMs),
            maxDurationMs: inNumberRange(maxDurationMs, defined.maxDurationMs),
            turnDelayMs: inNumberRange(turnDelayMs, defined.turnDelayMs),
        };
    }
} 

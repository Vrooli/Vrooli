import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

/** Increment when the schema of MessageConfigObject changes. */
const LATEST_MESSAGE_CONFIG_VERSION = "1.0";

/**
 * The result of a tool function call, which can be either a success or an error.
 */
export type ToolFunctionCallResult = {
    success: true;
    output: unknown;
} | {
    success: false;
    error: { code: string; message: string };
};

/**
 * Function call structure for tool usage
 */
export interface ToolFunctionCall {
    id: string;
    function: {
        name: string;
        arguments: string;
    };
    result?: ToolFunctionCallResult;
}

/**
 * Configuration for a single run within a chat message
 */
export interface ChatMessageRunConfig {
    runId: string;
    resourceVersionId: string;
    resourceVersionName?: string;
    taskId: string;
    runStatus?: string; // Using string to avoid circular dependency with RunStatus
    createdAt: string;
    completedAt?: string;
}

/**
 * Shape of the JSON blob stored in ConversationMessage.meta.
 * Keep it minimal—new optional fields can be added without migrations.
 */
export interface MessageConfigObject extends BaseConfigObject {
    /** Optional hints that can be injected into the next context window. */
    contextHints?: string[];
    /**
     * Arbitrary topic string that upstream services can subscribe to
     * (e.g. "payment-succeeded", "daily-summary").
     */
    eventTopic?: string;
    /**
     * Which bots should respond to this message.
     * Use "@all" to broadcast to every bot in the conversation.
     */
    respondingBots?: Array<string | "@all">;
    /** Author's role at the time this message was created. */
    role?: "user" | "assistant" | "system" | "tool";
    /**
     * Turn number this message belongs to.
     * Persisted so we can resume a crashed ConversationLoop exactly
     * where it left off.
     */
    turnId?: number | null;
    /**
     * Tool calls made by the assistant.
     * Each tool call has a function name, arguments, and optional result.
     */
    toolCalls?: ToolFunctionCall[];
    /**
     * Run configurations for routine executions triggered by this message.
     * Each entry represents a routine that was started from this message.
     */
    runs?: ChatMessageRunConfig[];
}

/**
 * Encapsulates message-level metadata with helpers for
 * (de)serialization, defaulting, and safe mutation.
 */
export class MessageConfig extends BaseConfig<MessageConfigObject> {
    contextHints?: MessageConfigObject["contextHints"];
    eventTopic?: MessageConfigObject["eventTopic"];
    respondingBots?: MessageConfigObject["respondingBots"];
    role?: MessageConfigObject["role"];
    turnId?: MessageConfigObject["turnId"];
    toolCalls?: MessageConfigObject["toolCalls"];
    runs?: MessageConfigObject["runs"];

    /* ---------------------------------------------------------------------- */
    /*  Constructors / Static helpers                                         */
    /* ---------------------------------------------------------------------- */

    constructor({ config }: { config: MessageConfigObject }) {
        super(config);
        this.contextHints = config.contextHints;
        this.eventTopic = config.eventTopic;
        this.respondingBots = config.respondingBots;
        this.role = config.role;
        this.turnId = config.turnId;
        this.toolCalls = config.toolCalls;
        this.runs = config.runs;
    }

    /**
     * Parse and instantiate a MessageConfig from a serialized string.
     * If useFallbacks is true, missing fields are populated with sensible
     * defaults to avoid null-checks downstream.
     */
    static parse(
        version: { config: MessageConfigObject },
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): MessageConfig {
        return super.parseBase<MessageConfigObject, MessageConfig>(
            version.config,
            logger,
            (cfg) => {
                if (opts?.useFallbacks ?? true) {
                    cfg.contextHints ??= [];
                    cfg.respondingBots ??= [];
                    cfg.role ??= "user";
                    cfg.turnId ??= null;
                    cfg.toolCalls ??= [];
                    cfg.runs ??= [];
                }
                return new MessageConfig({ config: cfg });
            },
        );
    }

    /**
     * Factory for a pristine MessageConfig with default values.
     * Useful when inserting new conversation messages.
     */
    static default(): MessageConfig {
        const config: MessageConfigObject = {
            __version: LATEST_MESSAGE_CONFIG_VERSION,
            resources: [],
            contextHints: [],
            respondingBots: [],
            role: "user",
            turnId: null,
            toolCalls: [],
            runs: [],
        };
        return new MessageConfig({ config });
    }

    /* ---------------------------------------------------------------------- */
    /*  Export & mutation helpers                                              */
    /* ---------------------------------------------------------------------- */

    /** Serializes the current state back to a plain object for storage. */
    override export(): MessageConfigObject {
        return {
            ...super.export(),
            contextHints: this.contextHints,
            eventTopic: this.eventTopic,
            respondingBots: this.respondingBots,
            role: this.role,
            turnId: this.turnId,
            toolCalls: this.toolCalls,
            runs: this.runs,
        };
    }

    /** Convenience mutators (type-safe; more can be added as needed). */
    setContextHints(hints: string[]): void {
        this.contextHints = hints;
    }
    setEventTopic(topic?: string): void {
        this.eventTopic = topic;
    }
    setRespondingBots(bots: Array<string | "@all">): void {
        this.respondingBots = bots;
    }
    setRole(role: MessageConfigObject["role"]): void {
        this.role = role;
    }
    setTurnId(turnId: number | null): void {
        this.turnId = turnId;
    }
    setToolCalls(toolCalls: ToolFunctionCall[]): void {
        this.toolCalls = toolCalls;
    }
    setRuns(runs: ChatMessageRunConfig[]): void {
        this.runs = runs;
    }
}

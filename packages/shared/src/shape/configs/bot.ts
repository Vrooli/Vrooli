import { type User } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import type { RunState } from "../../execution/routine.js";
import { type TranslationFunc, type TranslationKeyService } from "../../types.d.js";
import { BaseConfig, type BaseConfigObject, type ModelConfig } from "./base.js";

const LATEST_CONFIG_VERSION = "1.0";

// Agent specification types

/**
 * Agent-specific prompt configuration for enhanced behavior control.
 * Replaces the formal norms system with flexible natural language instructions.
 */
export interface AgentPromptConfig {
    /** How the agent prompt interacts with the default system prompt */
    mode: "supplement" | "replace";
    /** Where the prompt content comes from */
    source: "direct" | "resource";
    /** Direct prompt content (used when source is "direct") */
    content?: string;
    /** Resource ID containing the prompt (used when source is "resource") */
    resourceId?: string;
    /** Custom variable mappings for template substitution */
    variables?: Record<string, string>;
}

/**
 * Resource specification that can be accessed by an agent during execution.
 * Resources can come from the bot's configuration, swarm context, or global platform.
 */
export interface ResourceSpec {
    /** Category of resource that determines how it's accessed and used */
    type: "routine" | "document" | "tool" | "blackboard" | "link";
    /** Human-readable name for the resource */
    label: string;
    /** Unique identifier for the resource (optional for simple references) */
    id?: string;
    /** Access permissions for this resource (e.g., ["read", "write", "execute", "emit"]) */
    permissions?: string[];
    /** Source of the resource for precedence resolution */
    source?: "bot" | "swarm" | "global";
    /** Scope/path within the resource for granular access control (e.g., "metrics.*", "alerts.critical") */
    scope?: string;
    /** Optional description of what the resource provides */
    description?: string;
    /** Type-specific configuration options */
    config?: Record<string, any>;
}

/**
 * Defines a specific behavior that an agent will exhibit when triggered by events.
 * Behaviors are the core mechanism for emergent swarm coordination.
 */
export interface BehaviourSpec {
    /** Event matching and conditional activation configuration */
    trigger: {
        /** Event topic that this behavior responds to (must be from predefined platform events) */
        topic: string;
        /**
         * Optional jexl expression to evaluate when the trigger should be activated.
         * If omitted, the behavior triggers on every matching event.
         * 
         * Available context variables:
         * - event.data.* - Event payload fields
         * - event.type - Event type string
         * - event.timestamp - Event timestamp
         * - swarm.state - Current swarm execution state
         * - swarm.resources.* - Resource allocation info
         * - swarm.agents - Number of active agents
         * - bot.id - Current bot/agent ID
         * - bot.performance.* - Agent performance metrics
         * 
         * Examples:
         * - "event.data.remaining < event.data.allocated * 0.1"
         * - "event.data.newState == 'FAILED' || event.data.newState == 'TERMINATED'"
         * - "swarm.resources.consumed.credits > swarm.resources.allocated.credits * 0.8"
         */
        when?: string;
        /**
         * Progression control - determines what happens after this bot processes the event.
         * This replaces the ambiguous "handled" concept with clear semantics.
         */
        progression?: {
            /** 
             * How this bot's action affects event flow:
             * - "continue": Processing successful, proceed with event flow
             * - "block": Issue detected, halt event progression  
             * - "conditional": Use JEXL expression to determine progression
             * Default: "continue"
             */
            control?: "continue" | "block" | "conditional";
            /**
             * For conditional control, JEXL expression that returns boolean.
             * True = continue, False = block
             * Context includes result from action execution.
             * Example: "result.safe === true && result.confidence > 0.8"
             */
            condition?: string;
            /**
             * Should other bots be prevented from processing this event?
             * Default: false (allows multiple bots to process)
             */
            exclusive?: boolean;
        };
    };
    /** Action to execute when the trigger conditions are met */
    action: RoutineAction | InvokeAction | EmitAction;
    /** Quality of service level for message delivery (0=fire-and-forget, 1=at-least-once, 2=exactly-once) */
    qos?: 0 | 1 | 2;
}

/**
 * Action that executes a predefined routine when triggered.
 * Use for deterministic, repeatable behaviors with structured inputs/outputs.
 */
export interface RoutineAction {
    type: "routine";
    /** Human-readable routine name (used for resolution and display) */
    label: string;
    /** Optional direct routine ID for faster execution (resolved from label if not provided) */
    routineId?: string;
    /** Mapping of routine input variables to values from trigger context (e.g., {"goalId": "event.data.goalId"}) */
    inputMap?: Record<string, string>;
    /** Context-specific blackboard operations for routine outputs */
    outputOperations?: {
        /** Append routine outputs to blackboard arrays */
        append?: Array<{
            /** Field from routine's output (supports dot notation, e.g., "result.items") */
            routineOutput: string;
            /** Blackboard array to append to */
            blackboardId: string;
        }>;
        /** Increment blackboard numbers */
        increment?: Array<{
            /** Number from routine's output (supports dot notation, e.g., "stats.count") */
            routineOutput: string;
            /** Blackboard counter to increment */
            blackboardId: string;
        }>;
        /** Merge routine objects into blackboard (shallow merge) */
        merge?: Array<{
            /** Object from routine's output (supports dot notation, e.g., "config.settings") */
            routineOutput: string;
            /** Blackboard object to merge into */
            blackboardId: string;
        }>;
        /** Deep merge routine objects into blackboard (recursive merge) */
        deepMerge?: Array<{
            /** Object from routine's output (supports dot notation, e.g., "nested.config") */
            routineOutput: string;
            /** Blackboard object to deep merge into */
            blackboardId: string;
        }>;
        /** Simple assignment to blackboard (overwrites existing values) */
        set?: Array<{
            /** Any routine output (supports dot notation, e.g., "response.data") */
            routineOutput: string;
            /** Blackboard key to set */
            blackboardId: string;
        }>;
    };
}

/**
 * Action that invokes the agent's reasoning capabilities for adaptive responses.
 * Use when the agent needs to think, analyze, or make contextual decisions.
 */
export interface InvokeAction {
    type: "invoke";
    /** Description of what the agent should accomplish through reasoning */
    purpose: string;
}

/**
 * Action that emits a new event into the event bus system.
 * Use to trigger other agents' behaviors or coordinate swarm activities.
 */
export interface EmitAction {
    type: "emit";
    /** The event type to emit (must follow platform event patterns) */
    eventType: string;
    /** JEXL expressions mapping trigger context to event data fields */
    dataMapping?: Record<string, string>;
    /** Optional metadata for event delivery */
    metadata?: {
        /** Priority level for event processing */
        priority?: "low" | "medium" | "high" | "critical";
        /** Delivery guarantee level */
        deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
        /** Time-to-live for the event in milliseconds */
        ttl?: number;
    };
    /** Context-specific blackboard operations for emit outputs */
    outputOperations?: {
        /** Append emit outputs to blackboard arrays */
        append?: Array<{
            /** Field from emit result (supports dot notation, e.g., "metadata.priority", "responseData.items") */
            emitOutput: string;
            /** Blackboard array to append to */
            blackboardId: string;
        }>;
        /** Increment blackboard numbers */
        increment?: Array<{
            /** Number from emit result (supports dot notation, e.g., "metadata.retryCount") */
            emitOutput: string;
            /** Blackboard counter to increment */
            blackboardId: string;
        }>;
        /** Merge emit objects into blackboard (shallow merge) */
        merge?: Array<{
            /** Object from emit result (supports dot notation, e.g., "metadata", "responseData.config") */
            emitOutput: string;
            /** Blackboard object to merge into */
            blackboardId: string;
        }>;
        /** Deep merge emit objects into blackboard (recursive merge) */
        deepMerge?: Array<{
            /** Object from emit result (supports dot notation, e.g., "responseData.nested") */
            emitOutput: string;
            /** Blackboard object to deep merge into */
            blackboardId: string;
        }>;
        /** Simple assignment to blackboard (overwrites existing values) */
        set?: Array<{
            /** Any emit output (supports dot notation, e.g., "eventId", "timestamp", "metadata.deliveryStatus") */
            emitOutput: string;
            /** Blackboard key to set */
            blackboardId: string;
        }>;
    };
}

/**
 * Context available in jexl trigger expressions and input mappings.
 * Provides access to event data and swarm state based on agent's ResourceSpec permissions.
 */
export interface TriggerContext {
    /** Event information */
    event: {
        type: string;
        data: Record<string, any>;
        timestamp: Date;
        metadata?: Record<string, any>;
    };
    /** Swarm state and resource information */
    swarm: {
        state: RunState;
        resources: {
            allocated: { credits: number; tokens: number; time: number };
            consumed: { credits: number; tokens: number; time: number };
            remaining: { credits: number; tokens: number; time: number };
        };
        agents: number;
        id: string;
    };
    /** Bot's own performance metrics */
    bot: {
        id: string;
        performance: {
            tasksCompleted: number;
            tasksFailed: number;
            averageCompletionTime: number;
            successRate: number;
            resourceEfficiency: number;
        };
    };
    /** Result from action execution (available in progression.condition expressions) */
    result?: any;
    /** Primary objective of the swarm (read-only) */
    goal?: string;
    /** Sub-tasks within the swarm (filtered by agent's permissions) */
    subtasks?: Array<{
        id: string;
        description: string;
        status: string;
        assignee_bot_id?: string;
        priority?: string;
    }>;
    /** Blackboard data (filtered by agent's ResourceSpec scope permissions) */
    blackboard?: Record<string, any>;
    /** Event history records (filtered by agent's permissions) */
    records?: Array<{
        id: string;
        routine_name: string;
        created_at: string;
        caller_bot_id: string;
    }>;
    /** Event subscription mappings (read-only) */
    eventSubscriptions?: Record<string, string[]>;
    /** Swarm statistics (read-only) */
    stats?: {
        totalToolCalls: number;
        totalCredits: string;
        startedAt: number | null;
        lastProcessingCycleEndedAt: number | null;
    };
}

/**
 * Complete specification for an autonomous agent within a swarm.
 * Defines the agent's purpose, behaviors, prompt configuration, and resource access.
 * Stored in bot configuration for reuse across different swarms.
 */
export interface AgentSpec {
    /** Primary objective or purpose that guides the agent's decision-making */
    goal?: string;
    /** Functional role within the swarm (e.g., "coordinator", "specialist", "monitor", "bridge") */
    role?: string;
    /** List of event topics the agent subscribes to (must be from predefined platform events) */
    subscriptions?: string[];
    /** Event-driven behaviors that define how the agent responds to different situations */
    behaviors?: BehaviourSpec[];
    /** Agent-specific prompt configuration that replaces formal norms */
    prompt?: AgentPromptConfig;
    /** Resources available to the agent during execution */
    resources?: ResourceSpec[];
}

export interface BotConfigObject extends BaseConfigObject {
    modelConfig?: ModelConfig;
    maxTokens?: number;
    agentSpec?: AgentSpec;
}

export class BotConfig extends BaseConfig<BotConfigObject> {
    modelConfig?: BotConfigObject["modelConfig"];
    maxTokens?: BotConfigObject["maxTokens"];
    agentSpec?: BotConfigObject["agentSpec"];

    constructor({ botSettings }: { botSettings: BotConfigObject }) {
        super({ config: botSettings });
        this.modelConfig = botSettings.modelConfig;
        this.maxTokens = botSettings.maxTokens;
        this.agentSpec = botSettings.agentSpec;
    }

    static parse(
        bot: Pick<User, "botSettings"> | null | undefined,
        logger: PassableLogger,
    ): BotConfig {
        const botSettings = bot?.botSettings;
        return super.parseBase<BotConfigObject, BotConfig>(
            botSettings,
            logger,
            ({ config }) => {
                // agentSpec is optional and doesn't need defaults
                return new BotConfig({ botSettings: config });
            },
        );
    }

    static default(): BotConfig {
        return new BotConfig({
            botSettings: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                modelConfig: undefined,
                maxTokens: undefined,
                agentSpec: undefined,
            },
        });
    }

    export(): BotConfigObject {
        return {
            ...super.export(),
            modelConfig: this.modelConfig,
            maxTokens: this.maxTokens,
            agentSpec: this.agentSpec,
        };
    }
}

export type LlmModel = {
    name: TranslationKeyService,
    description?: TranslationKeyService,
    value: string,
};

export function getModelName(option: LlmModel | null, t: TranslationFunc) {
    return option ? t(option.name, { ns: "service" }) : "";
}
export function getModelDescription(option: LlmModel, t: TranslationFunc) {
    return option && option.description ? t(option.description, { ns: "service" }) : "";
}

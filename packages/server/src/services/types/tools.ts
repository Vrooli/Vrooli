import type { BlackboardItem, ChatConfigObject, McpToolName, ModelType, ResourceSubType, SwarmPolicy, SwarmResource, SwarmSubTask, TeamConfigObject, ResourceType as TypeOfResource } from "@vrooli/shared";

/** A graph or sequence of routines */
type RoutineMultiStep = ResourceSubType.RoutineMultiStep;
/** Perform an action within Vrooli (e.g. 'create a new note') */
type RoutineInternalAction = ResourceSubType.RoutineInternalAction;
/** Call an API */
type RoutineApi = ResourceSubType.RoutineApi;
/** Run data conversion code */
type RoutineCode = ResourceSubType.RoutineCode;
/** Data (for passing into other routines) */
type RoutineData = ResourceSubType.RoutineData;
/** Generative AI (e.g. LLM, image generation) */
type RoutineGenerate = ResourceSubType.RoutineGenerate;
/** Contains only context information/instructions/resources */
type RoutineInformational = ResourceSubType.RoutineInformational;
/** Run a smart contract */
type RoutineSmartContract = ResourceSubType.RoutineSmartContract;
/** Perform a web search */
type RoutineWeb = ResourceSubType.RoutineWeb;
/** An autonomous agent */
type Bot = "Bot";
/** Team of bots and/or humans */
type Team = ModelType.Team;
/** AI prompt */
type StandardPrompt = ResourceSubType.StandardPrompt;
/** Defines a standard data schema, used for validating routine inputs/outputs and ensuring interoperability. */
type StandardDataStructure = ResourceSubType.StandardDataStructure;
/** Group resources together */
type Project = TypeOfResource.Project;
/** Store short memos/thoughts/etc. */
type Note = TypeOfResource.Note;
/** KV bucket, table, S3 object, vector index, etc. */
type ExternalData = "ExternalData";
type ResourceType = RoutineMultiStep
    | RoutineInternalAction
    | RoutineApi
    | RoutineCode
    | RoutineData
    | RoutineGenerate
    | RoutineInformational
    | RoutineSmartContract
    | RoutineWeb
    | Bot
    | Team
    | StandardPrompt
    | StandardDataStructure
    | Project
    | Note
    | ExternalData;

/**
 * Return the detailed input schema for ResourceManage given a resource variant *and* CRUD op.
 */
export interface DefineToolParams {
    toolName: McpToolName.ResourceManage;
    /** Resource type */
    variant: ResourceType;
    /** The operation to define */
    op: "add" | "find" | "update" | "delete";
}

/** Direct message */
type RecipientBot = { kind: "bot"; botId: string }
/** Direct DM */
type RecipientUser = { kind: "user"; userId: string }
/** Whole thread */
type RecipientChat = { kind: "chat"; chatId: string }
/** MQTT / role broadcast */
type RecipientTopic = { kind: "topic"; topic: string };
export type Recipient = RecipientBot
    | RecipientUser
    | RecipientChat
    | RecipientTopic;
/**
 * Send a chat message or publish an event for agents to react to.
 */
export interface SendMessageParams {
    /** Where the message goes. */
    recipient: Recipient;
    /** Plain-text or structured JSON payload. */
    content: string | Record<string, unknown>;
    /** Optional extra headers (JSON-serialisable). */
    metadata?: Record<string, unknown>;
    /** Normal vs high-priority routing or QoS level (default: normal). */
    priority?: "normal" | "high" | 0 | 1;
}


/**
 * Find Resource
 */
export type ResourceManageFindParams = {
    op: "find";
    resource_type: ResourceType
    id?: string;
    query?: string;
    /** Call `DefineTool` with `op = find` and `resource_type` to the target resource type to get the available filters. */
    filters?: Record<string, any>;
}
/**
 * Add Resource
 */
export type ResourceManageAddParams = {
    op: "add";
    resource_type: ResourceType
    /** Call `DefineTool` with `op = add` and `resource_type` to the target resource type to get the available attributes. */
    attributes: Record<string, any>;
}
/**
 * Update Resource
 */
export type ResourceManageUpdateParams = {
    op: "update";
    resource_type: ResourceType
    id: string;
    /** Call `DefineTool` with `op = update` and `resource_type` to the target resource type to get the available attributes. */
    attributes: Record<string, any>;
}
/**
 * Delete Resource
 */
export type ResourceManageDeleteParams = {
    op: "delete";
    resource_type: ResourceType;
    id: string;
}
/**
 * Find, add, update or delete a resource depending on 'op'.
 */
export type ResourceManageParams = ResourceManageFindParams | ResourceManageAddParams | ResourceManageUpdateParams | ResourceManageDeleteParams;


/** Assign routine to a single bot using their ID */
type ActorSelectionBot = { actorId: string }
/** Map routine's defined roles to bot IDs that will fill them */
type ActorSelectionRole = { actorMap: Record<string, string> }
/** Single bot ID or a role→bot map (use one, not both) */
type ActorSelection = ActorSelectionBot | ActorSelectionRole;
export type RunRoutineParamsBase = {
    /** sync (blocking) | async (background); default = sync */
    mode?: "sync" | "async";
    /** Only when mode==="async": ISO-8601 instant OR cron pattern */
    schedule?: { at?: string; cron?: string; timezone?: string };
    /** 
     * Normal vs high-priority routing or QoS level (default: normal). 
     * 
     * NOTE: High-priority runs may jump the queue, but will cost slightly more.
     */
    priority?: "low" | "normal" | "high";
}
/** Start a new run */
export type RunRoutineStart = RunRoutineParamsBase & ActorSelection & {
    action: "start";
    /** Routine to execute */
    routineId: string;
    /** Inputs to pass to the routine */
    inputs?: Record<string, unknown>;
}
/** Manage an existing run */
export type RunRoutineManage = RunRoutineParamsBase & {
    action: "pause" | "resume" | "cancel" | "status";
    /** Existing run identifier - NOT the routine ID */
    runId: string;
}
/**
 * Start or manage a routine run.
 */
export type RunRoutineParams = RunRoutineStart | RunRoutineManage;


/** Reservation taken from caller's remaining quota */
export interface Reservation {
    /** Percentage of credits to reserve (1-100) */
    creditsPct: number;
    /** Percentage of messages to reserve (1-100) */
    messagesPct?: number;
    /** Percentage of duration to reserve (1-100) */
    durationPct?: number;
}
/** Minimal child swarm (single bot, inherit defaults) */
export interface SpawnSwarmSimple {
    kind: "simple";
    /** The primary objective of the child swarm */
    goal: string;
    /** ID of the bot that will lead the child swarm */
    swarmLeader: string;
    /** How many credits/messages/time to reserve for the child swarm */
    reservation?: Reservation;
    /** Context to pass to the child swarm */
    resources?: SwarmResource[];
}
/** Rich child swarm: multi-bot, custom event map, seed subtasks */
export interface SpawnSwarmRich {
    kind: "rich";
    /** The primary objective of the child swarm */
    goal: string;
    /** The team to use for the child swarm */
    teamId: string;
    /** How many credits/messages/time to reserve for the child swarm */
    reservation?: Reservation;
    /** Context to pass to the child swarm */
    resources?: SwarmResource[];
    /** Subtasks to start the child swarm with */
    subtasks?: SwarmSubTask[];
    /** Map of event topics to bot IDs that should respond to them */
    eventSubscriptions?: Record<string, string[]>;
    /** Visibility of the child swarm */
    policy?: SwarmPolicy;
}
/**
 * Create a new AI agent swarm.
 */
export type SpawnSwarmParams = SpawnSwarmSimple | SpawnSwarmRich;

type Patch<T> = {
    /** New or updated items */
    set?: T[];
    /** IDs to remove */
    delete?: string[];
};

/**
 * Extended patch operations for blackboard that support accumulation
 */
export type BlackboardPatch = {
    /** New or updated items (overwrites existing values) */
    set?: BlackboardItem[];
    /** IDs to remove */
    delete?: string[];
    /** Append values to existing arrays */
    append?: Array<{
        /** ID of the blackboard item to append to */
        id: string;
        /** Values to append (target must be an array) */
        values: unknown[];
    }>;
    /** Increment numeric values */
    increment?: Array<{
        /** ID of the blackboard item to increment */
        id: string;
        /** Amount to increment by (can be negative for decrement) */
        amount: number;
    }>;
    /** Merge objects (shallow merge) */
    merge?: Array<{
        /** ID of the blackboard item to merge with */
        id: string;
        /** Object to merge (target must be an object) */
        object: Record<string, unknown>;
    }>;
    /** Deep merge objects (recursive merge) */
    deepMerge?: Array<{
        /** ID of the blackboard item to deep merge with */
        id: string;
        /** Object to deep merge (target must be an object) */
        object: Record<string, unknown>;
    }>;
};
/**
 * Updates shared state for the current swarm. Bots in a swarm use this to coordinate and manage resources and responsibilities.
 */
export type UpdateSwarmSharedStateParams = Pick<ChatConfigObject, "goal" | "subtaskLeaders" | "teamId"> & {
    /**
     * Mutations to the swarm's canonical task list
     * - `set` inserts new tasks or overwrites existing ones
     * - `delete` removes tasks by ID
     */
    subTasks?: Patch<SwarmSubTask>;
    /**
     * Mutations to the shared resource registry
     * - lets any bot publish or revoke artefacts
     */
    resources?: Patch<SwarmResource>;
    /** 
     * Scratchpad for storing arbitrary data that is shared between bots.
     * Supports both basic operations (set/delete) and accumulation operations
     * (append, increment, merge) for collaborative workflows.
     */
    blackboard?: BlackboardPatch;
    /**
     * Update event map. topic → subscriber bot IDs
     */
    eventSubscriptions?: {
        /** Map of new or updated items */
        set?: Record<string, string[]>;
        /** Keys to remove */
        delete?: string[];
    }
    /**
     * Updates to the team configuration (organizational structure, roles, etc.)
     * This will update the team object in the database and refresh the team config in the swarm
     */
    teamConfig?: TeamConfigObject;
}

/**
 * End the swarm. Called when the goal is complete or limits are reached.
 */
export interface EndSwarmParams {
    /**
     * How to end the session:
     *  • "graceful" - allow bots to finish their current turn  
     *  • "force"    - cancel immediately. May interrupt routines
     * 
     * @default "graceful"
     */
    mode?: "graceful" | "force";
    /** Reason for ending; sent to participants & saved to the audit trail. */
    reason?: string;
}

/**
 * Available routine-level tools, ordered by (likely) frequency of use.
 * 
 * These are only available when the MCP server is set up for a specific routine, 
 * rather than the site as a whole
 */
export enum McpRoutineToolName {
    /** Start a routine */
    StartRoutine = "start_routine",
    /** Stop a routine */
    StopRoutine = "stop_routine",
}

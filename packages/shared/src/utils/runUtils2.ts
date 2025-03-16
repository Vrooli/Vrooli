/**
 * Contains logic for running routines and projects, both in the server and in the client. 
 * Specific implementations for fetching and saving data are abstracted out, and must be provided
 * by the caller.
 */
import { DbObject, ModelType, ProjectVersion, ProjectVersionDirectory, RoutineVersion, RunRoutineStep, RunStatus } from "../api/types";
import { API_CREDITS_MULTIPLIER } from "../consts/api";
import { PassableLogger } from "../consts/commonTypes";
import { DOLLARS_1_CENTS, MINUTES_5_MS } from "../consts/numbers";
import { RoutineVersionConfig } from "./routineConfig";
import { RoutineGraphType } from "./routineGraph";

/** Limit routines to $1 for now */
export const DEFAULT_MAX_RUN_CREDITS = BigInt(DOLLARS_1_CENTS) * API_CREDITS_MULTIPLIER;
/** Limit routines to 5 minutes for now */
export const DEFAULT_MAX_RUN_TIME = MINUTES_5_MS;

/**
 * A location in the run.
 * 
 * Contains enough information to be queriable directly, without having to 
 * load previous locations in the stack.
 */
export type Location = {
    /** The type of object */
    __typename: ModelType.RoutineVersion | ModelType.ProjectVersion | ModelType.ProjectVersionDirectory;
    /** The object ID */
    objectId: string;
    /** 
     * The ID of the location within the object.
     * Typically this is a node ID within a routine
     */
    locationId: string;
};

/**
 * A stack representing the current location in the run.
 * 
 * Each time a nested graph is encountered, a new entry is added to the stack.
 * Each time a nested graph is exited, the last entry is removed from the stack.
 */
export type LocationStack = Location[];

/**
 * Behavior when a run reaches a limit.
 */
export type RunLimitBehavior = "Pause" | "Stop";

/**
 * Limits for a run request, as well as the behavior when those limits are reached. 
 * 
 * NOTE 1: These should be reset when restarting a failed run, but not when resuming a paused run.
 * NOTE 2: Just because a limit is set doesn't mean it will be reached. If a user has 100 credits,
 * but the limit is set to 10_000, the run will stop at 100 credits.
 */
export type RunRequestLimits = {
    /** 
     * Maximum tokens in routine context, which accumulates relevant info 
     * over multiple steps. Having fewer tokens reduces the cost of 
     * generating LLM responses.
     */
    maxContextTokens?: number;
    /** Maximum number of credits that can be spent */
    maxCredits?: string;
    /** Maximum number of steps that can be run */
    maxSteps?: number;
    /** Maximum time that an be spent on the run, in milliseconds */
    maxTime?: number;
    /** What to do on max credits reached */
    onMaxCredits?: RunLimitBehavior;
    /** What to do in max time reached */
    onMaxTime?: RunLimitBehavior;
    /** What to do on max steps reached */
    onMaxSteps?: RunLimitBehavior;
}

/**
 * Represents the accumulated data in the current run.
 */
export type RunMetrics = {
    /** How much time (in milliseconds) has elapsed in all steps combined */
    timeElapsed: number;
    /** How many steps (excluding the current step) have been run so far. */
    stepsRun: number;
    /** How many tokens are in the routine context */
    contextTokens: number;
    /** How many credits have been spent so far */
    creditsSpent: string;
}

/**
 * Represents the current run progress.
 */
export interface RunProgress {
    /** The current location in the run */
    location: LocationStack;
    /** Previous step history */
    steps: RunRoutineStep[];
    /** Accumulated metrics (cost, time, etc.) */
    metrics: RunMetrics;
    /** The current overall run status */
    status: RunStatus;
}

/**
 * Interface for navigating a run step-by-step. Note that we don't need methods 
 * for locating the current step, as that can be accomplished using the loader. 
 * And we also don't need methods for finding the previous step, as the state machine 
 * handles popping the location stack until a new next step is found.
 * 
 * Should be implemented by each supported workflow type (e.g. BPMN, DMN)
 */
export interface IRunStepNavigator {
    /**
     * Return the next nodes, given the current location.
     * 
     * NOTE 1: This does not traverse up the stack to find the next location. 
     * It simply checks if there are any next nodes in the current graph.
     * 
     * NOTE 2: If there are edges to nodes with conditions (e.g. bpmn:SequenceFlow.conditionExpression), any edge which 
     * evaluates to false should not be included in the result.
     * 
     * @param object The current routine/project/project directory object
     * @param location The current location (last item in the location stack) in the current routine/project/project directory
     * @param context The current context (e.g. variables, state) for the run, which may be needed to evaluate conditions
     * @returns The next location(s), or none if at the end of the current routine/project/project directory
     */
    getNextNodes<T extends RoutineVersion | ProjectVersion | ProjectVersionDirectory>(object: T, location: Location, context: unknown): Location[];
    /**
     * If the current node is something we can step into (e.g. a multi-step routine, a project, or a project directory),
     * return the start nodes for that sub-routine or sub-project.
     * 
     * For multi-step routines, the start nodes are typically clearly defined.
     * For projects, every immediate directory is a potential start node.
     * For project directories, every immediate subdirectory or file is a potential start node.
     * 
     * @param object The current object loaded from the location
     * @param context The current context (e.g. variables, state) for the run, which may be needed to evaluate conditions
     * @returns The start nodes (starting locations) for the sub-routine or sub-project, or none if the current node is a leaf node
     */
    getStartNodes<T extends RoutineVersion | ProjectVersion | ProjectVersionDirectory>(object: T, context: unknown): Location[];
}

export type NavigatorRegistry = {
    [graphType in RoutineGraphType]: IRunStepNavigator;
}

/**
 * A simple factory that returns the right navigator given a routine type.
 */
export class NavigatorFactory {
    constructor(private readonly registry: NavigatorRegistry) { }

    public getNavigator(graphType: RoutineGraphType): IRunStepNavigator {
        const nav = this.registry[graphType];
        if (!nav) throw new Error(`No navigator found for ${graphType}`);
        return nav;
    }
}

class BpmnNavigator implements IRunStepNavigator {
    public getNextNodes(object: unknown, location: Location, context: unknown): Location[] {
        //TODO implement
        return [];
    }
    public getStartNodes(object: unknown): Location[] {
        //TODO implement
        return [];
    }
}

/**
 * Handles loading routine and project information required for running a routine or project.
 * 
 * Initially fetches the data from the server, with subsequent calls using the cache.
 */
export abstract class RunLoader {
    /**
     * Map of loaded routine versions, keyed by routine ID.
     */
    protected routineCache: Map<string, RoutineVersion> = new Map();
    /**
     * Map of loaded project versions, keyed by project ID.
     */
    protected projectCache: Map<string, ProjectVersion> = new Map();

    /**
     * Fetch the object for a given location.
     * 
     * @param location The location to fetch
     * @returns The fetched object, or null if not found
     */
    public abstract fetchLocation<T extends DbObject<"ProjectVersion" | "RoutineVersion">>(location: Location): Promise<T | null>;

    /**
     * Load the object for a given location, and store it in the cache.
     * 
     * @param location The location to load
     * @returns The loaded object, or null if not found
     */
    public async loadLocation<T extends DbObject<"ProjectVersion" | "RoutineVersion">>(location: Location): Promise<T | null> {
        const cached = this.findInCache(location);
        if (cached) {
            return cached as unknown as T;
        }
        const loaded = await this.fetchLocation<T>(location);
        if (loaded) {
            this.addToCache(loaded);
        }
        return loaded;
    }

    /**
     * Load the full location stack, and store every object in the cache.
     * 
     * @param locationStack The location stack to load
     * @returns The last loaded object, or null if not found
     */
    public async loadLocationStack<T extends DbObject<"ProjectVersion" | "RoutineVersion">>(locationStack: LocationStack): Promise<T | null> {
        let current: T | null = null;
        for (const location of locationStack) {
            current = await this.loadLocation<T>(location);
        }
        return current;
    }

    /**
     * Called whenever the cache changes. Can be used to store the cache 
     * in redis or localStorage, so that it can be reloaded later without 
     * having to fetch everything again.
     */
    public abstract onCacheChange(): void;

    /**
     * Add an object to the relevant cache.
     * 
     * @param object The object to cache
     */
    private addToCache(object: DbObject<"ProjectVersion" | "RoutineVersion">): void {
        if (object.__typename === ModelType.RoutineVersion) {
            this.routineCache.set(object.id, object as RoutineVersion);
        } else if (object.__typename === ModelType.ProjectVersion) {
            this.projectCache.set(object.id, object as ProjectVersion);
        }
        this.onCacheChange();
    }

    /**
     * Finds the object for a given location in the cache.
     * 
     * @param location The location to find
     * @returns The cached object, or null if not found
     */
    private findInCache(location: Pick<Location, "__typename" | "objectId">): ProjectVersion | RoutineVersion | null {
        const cachedObject = location.__typename === ModelType.RoutineVersion
            ? this.routineCache.get(location.objectId)
            : this.projectCache.get(location.objectId);
        return cachedObject || null;
    }
}

/**
 * Handles saving and storing run progress.
 */
export abstract class RunPersistence {
    /**
     * Save the current run progress.
     * 
     * @param runId The ID for this run, or DUMMY_ID if creating a new run
     * @param progress The current progress object
     */
    public abstract saveProgress(runId: string, progress: RunProgress): Promise<void>;
    /**
     * Load the current run progress.
     * 
     * @param runId The ID for this run
     * @returns The loaded progress object, or null if not found
     */
    public abstract loadProgress(runId: string): Promise<RunProgress | null>;
}

/**
 * Handles making decisions when there are multiple possible next steps.
 * This allows for different strategies to be used at runtime, such as 
 * user choice, random selection, or LLM-based selection.
 */
export type DecisionStrategy = {
    choose: (validNextNodeIds: string[], context?: any) => Promise<string>;
};

/**
 * Simplest decision strategy: always pick the first valid next node.
 */
export const autoPickFirst: DecisionStrategy = {
    async choose(validNextNodeIds) {
        const nextNode = validNextNodeIds[0];
        if (!nextNode) {
            throw new Error("No valid next nodes");
        }
        return nextNode;
    },
};

/** Maps graph types to navigators */
const navigatorRegistry: NavigatorRegistry = {
    "BPMN-2.0": new BpmnNavigator(),
};

/** Factory for navigators. Allows traversal of any supported graph type. */
export const navigatorFactory = new NavigatorFactory(navigatorRegistry);

/**
 * High-level state machine for running a routine.
 * 
 * Orchestrates transitions between steps, handles drilling down 
 * into sub-routines, and updates the run progress.
 * 
 * NOTE: This is agnostic to the actual execution of steps and graph architecture/traversal, 
 * which should be handled by a separate service.
 */
export class RunStateMachine {
    /** Finds the appropriate navigator for stepping through the current routine graph */
    private navigatorFactory: NavigatorFactory;
    /** Loader for fetching routine information */
    private loader: RunLoader;
    /** Persistence service for saving/loading run progress */
    private persistence: RunPersistence;
    /** Strategy for making decisions when there are multiple possible next steps */
    private decisionStrategy: DecisionStrategy;
    /** Logger for debugging */
    private logger: PassableLogger;

    constructor(
        navigatorFactory: NavigatorFactory,
        loader: RunLoader,
        persistence: RunPersistence,
        decisionStrategy: DecisionStrategy,
        logger: PassableLogger,
    ) {
        this.navigatorFactory = navigatorFactory;
        this.loader = loader;
        this.persistence = persistence;
        this.decisionStrategy = decisionStrategy;
        this.logger = logger;
    }

    /**
     * Initialize a new run or resume an existing run from the database.
     * 
     * @param runId A unique identifier for this run
     * @param initialRoutineId The starting BPMN/routine ID (if new run)
     * @returns The loaded or created WorkflowProgress
     */
    public async initializeRun(
        runId: string,
        initialRoutineId: string,
    ): Promise<WorkflowProgress> {
        const existingProgress = await this.persistence.loadProgress(runId);

        if (existingProgress) {
            // Resume
            return existingProgress;
        } else {
            // Create fresh
            const newProgress: WorkflowProgress = {
                locationStack: [initialRoutineId],  // Start at top-level routine
                visitedNodes: {},
                metrics: {
                    cost: 0,
                    time: 0,
                },
                status: "Idle",
            };
            await this.persistence.saveProgress(runId, newProgress);
            return newProgress;
        }
    }

    /**
     * Execute the next step in the run.
     * @param locationStack The current location stack
     * @returns Updated location stack
     */
    public async executeNext(locationStack: LocationStack): Promise<LocationStack> {
        // 1. Look at top of stack:
        const currentLocation = locationStack[locationStack.length - 1];
        if (!currentLocation) {
            throw new Error("Location stack is empty");
        }

        // 2. Get the routine definition:
        // TODO might not always be a routine. Should support other types. Also might be null, so should handle that too
        const routineDef = await this.loader.loadLocation<RoutineVersion>(currentLocation);
        if (!routineDef) {
            throw new Error(`Routine not found: ${currentLocation.objectId}`);
        }

        // 3. Get the appropriate navigator:
        const routineConfig = RoutineVersionConfig.deserialize(routineDef, this.logger, { useFallbacks: true });
        const graphType = routineConfig.graph?.__type;
        if (!graphType) {
            throw new Error("Routine has no graph type");
        }
        const navigator = this.navigatorFactory.getNavigator(graphType);

        // 4. Find the next node(s):
        const nextNodes = navigator.getNextNodes<RoutineVersion>(routineDef, currentLocation, contextData);
        // If there are no next nodes, we're at the end of the current routine.
        if (nextNodes.length === 0) {
            // Pop the stack to go back to the parent
            locationStack.pop();
            return locationStack;
        }

        // 5. Determine which node to go to next
        let chosenNodeId: Location;
        if (nextNodes.length === 1) {
            // If exactly 1, no decision needed
            chosenNodeId = nextNodes[0];
        } else {
            // If there's more than 1, we need a decision
            chosenNodeId = await this.decisionStrategy.choose(nextNodes, contextData);
        }

        const startNodes = navigator.getStartNodes(routineDef, contextData);
        if (startNodes.length > 0) {
            let chosenStartNode: Location;
            if (startNodes.length === 1) {
                chosenStartNode = startNodes[0];
            } else {
                chosenStartNode = await this.decisionStrategy.choose(startNodes, contextData);
            }
            // sub-routine call → push onto stack
            locationStack.push(chosenStartNode); // or some start node ID
        } else {
            // same routine, just replace the top with the new nodeId
            currentLocation.nodeId = nextNodeId;
        }

        return locationStack;
    }

    /**
     * Switch how decisions are made at runtime.
     */
    public setDecisionStrategy(newStrategy: DecisionStrategy) {
        this.decisionStrategy = newStrategy;
    }
}

//TODO context building, concurrency management, better project handling, triggers, maxTime timeout
// TODO for conurrency, should be able to toggle on/off by navigator type. For BPMN, branching is handled using gateways. Make sure that we properly support each type of gateway.

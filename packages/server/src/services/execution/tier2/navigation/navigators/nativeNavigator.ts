import { type Logger } from "winston";
import {
    type Navigator,
    type Location,
    type StepInfo,
} from "@vrooli/shared";

/**
 * Native Vrooli routine definition format
 */
export interface NativeRoutineDefinition {
    version: string;
    steps: NativeStep[];
    edges: NativeEdge[];
    metadata?: {
        startNodeId?: string;
        endNodeIds?: string[];
    };
}

interface NativeStep {
    id: string;
    name: string;
    type: "action" | "decision" | "loop" | "parallel" | "subroutine";
    description?: string;
    config?: Record<string, unknown>;
    inputs?: Record<string, unknown>;
    outputs?: Record<string, unknown>;
}

interface NativeEdge {
    id: string;
    from: string;
    to: string;
    condition?: string; // JavaScript expression
    label?: string;
}

/**
 * NativeNavigator - Navigator for Vrooli's native workflow format
 * 
 * This navigator handles Vrooli's native routine definition format,
 * which is designed to be simple, flexible, and powerful. It supports:
 * 
 * - Sequential and parallel execution
 * - Conditional branching
 * - Loops and iterations
 * - Subroutine calls
 * - Dynamic path selection
 * 
 * The native format is optimized for AI-assisted creation and modification,
 * making it easy for both humans and AI agents to author workflows.
 */
export class NativeNavigator implements Navigator {
    readonly type = "native";
    readonly version = "1.0.0";
    
    private readonly logger: Logger;
    private definitionCache: Map<string, NativeRoutineDefinition> = new Map();

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Checks if this navigator can handle the given routine
     */
    canNavigate(routine: unknown): boolean {
        try {
            const def = routine as NativeRoutineDefinition;
            
            // Check required fields
            if (!def.version || !Array.isArray(def.steps) || !Array.isArray(def.edges)) {
                return false;
            }

            // Check version compatibility
            const majorVersion = parseInt(def.version.split(".")[0]);
            if (majorVersion !== 1) {
                return false;
            }

            // Basic structural validation
            if (def.steps.length === 0) {
                return false;
            }

            return true;

        } catch {
            return false;
        }
    }

    /**
     * Gets the starting location in the routine
     */
    getStartLocation(routine: unknown): Location {
        const def = this.validateAndCache(routine);
        
        // Use explicit start node if defined
        if (def.metadata?.startNodeId) {
            const startStep = def.steps.find(s => s.id === def.metadata!.startNodeId);
            if (!startStep) {
                throw new Error(`Start node ${def.metadata.startNodeId} not found`);
            }
            
            return this.createLocation(startStep.id, def);
        }

        // Find nodes with no incoming edges
        const nodesWithIncoming = new Set(def.edges.map(e => e.to));
        const startNodes = def.steps.filter(s => !nodesWithIncoming.has(s.id));

        if (startNodes.length === 0) {
            throw new Error("No start node found (circular graph)");
        }

        if (startNodes.length > 1) {
            this.logger.warn("[NativeNavigator] Multiple start nodes found, using first", {
                nodeIds: startNodes.map(n => n.id),
            });
        }

        return this.createLocation(startNodes[0].id, def);
    }

    /**
     * Gets the next possible locations from current location
     */
    getNextLocations(current: Location, context: Record<string, unknown>): Location[] {
        const def = this.definitionCache.get(current.routineId);
        if (!def) {
            throw new Error(`Routine ${current.routineId} not in cache`);
        }

        // Find outgoing edges from current node
        const outgoingEdges = def.edges.filter(e => e.from === current.nodeId);

        if (outgoingEdges.length === 0) {
            return []; // End of path
        }

        // Evaluate conditions and filter valid edges
        const validEdges = outgoingEdges.filter(edge => {
            if (!edge.condition) {
                return true; // No condition = always valid
            }

            try {
                // Safely evaluate condition
                const result = this.evaluateCondition(edge.condition, context);
                return !!result;
            } catch (error) {
                this.logger.error("[NativeNavigator] Failed to evaluate edge condition", {
                    edgeId: edge.id,
                    condition: edge.condition,
                    error: error instanceof Error ? error.message : String(error),
                });
                return false;
            }
        });

        // Convert to locations
        return validEdges.map(edge => this.createLocation(edge.to, def));
    }

    /**
     * Checks if location is an end location
     */
    isEndLocation(location: Location): boolean {
        const def = this.definitionCache.get(location.routineId);
        if (!def) {
            throw new Error(`Routine ${location.routineId} not in cache`);
        }

        // Use explicit end nodes if defined
        if (def.metadata?.endNodeIds) {
            return def.metadata.endNodeIds.includes(location.nodeId);
        }

        // Check if node has no outgoing edges
        const hasOutgoing = def.edges.some(e => e.from === location.nodeId);
        return !hasOutgoing;
    }

    /**
     * Gets information about a step at a location
     */
    getStepInfo(location: Location): StepInfo {
        const def = this.definitionCache.get(location.routineId);
        if (!def) {
            throw new Error(`Routine ${location.routineId} not in cache`);
        }

        const step = def.steps.find(s => s.id === location.nodeId);
        if (!step) {
            throw new Error(`Step ${location.nodeId} not found in routine`);
        }

        return {
            id: step.id,
            name: step.name,
            type: step.type,
            description: step.description,
            inputs: step.inputs,
            outputs: step.outputs,
            config: step.config,
        };
    }

    /**
     * Gets dependencies for a step
     */
    getDependencies(location: Location): string[] {
        const def = this.definitionCache.get(location.routineId);
        if (!def) {
            throw new Error(`Routine ${location.routineId} not in cache`);
        }

        // Find incoming edges
        const incomingEdges = def.edges.filter(e => e.to === location.nodeId);
        
        // Return source node IDs
        return incomingEdges.map(e => e.from);
    }

    /**
     * Gets parallel branches from a location
     */
    getParallelBranches(location: Location): Location[][] {
        const def = this.definitionCache.get(location.routineId);
        if (!def) {
            throw new Error(`Routine ${location.routineId} not in cache`);
        }

        const step = def.steps.find(s => s.id === location.nodeId);
        if (!step || step.type !== "parallel") {
            return [];
        }

        // Find all outgoing edges (these define the parallel branches)
        const outgoingEdges = def.edges.filter(e => e.from === location.nodeId);
        
        // Each edge starts a branch - trace to convergence
        const branches: Location[][] = [];
        
        for (const edge of outgoingEdges) {
            const branch = this.traceBranch(edge.to, def);
            branches.push(branch);
        }

        return branches;
    }

    /**
     * Private helper methods
     */
    private validateAndCache(routine: unknown): NativeRoutineDefinition {
        if (!this.canNavigate(routine)) {
            throw new Error("Invalid native routine definition");
        }

        const def = routine as NativeRoutineDefinition;
        
        // Validate graph structure
        this.validateGraphStructure(def);

        // Cache for future use (using first step ID as routine ID for now)
        const routineId = def.steps[0].id; // TODO: Use proper routine ID
        this.definitionCache.set(routineId, def);

        return def;
    }

    private validateGraphStructure(def: NativeRoutineDefinition): void {
        const stepIds = new Set(def.steps.map(s => s.id));

        // Validate edges reference valid steps
        for (const edge of def.edges) {
            if (!stepIds.has(edge.from)) {
                throw new Error(`Edge ${edge.id} references unknown source step: ${edge.from}`);
            }
            if (!stepIds.has(edge.to)) {
                throw new Error(`Edge ${edge.id} references unknown target step: ${edge.to}`);
            }
        }

        // Check for duplicate step IDs
        if (stepIds.size !== def.steps.length) {
            throw new Error("Duplicate step IDs found");
        }
    }

    private createLocation(nodeId: string, def: NativeRoutineDefinition): Location {
        const routineId = def.steps[0].id; // TODO: Use proper routine ID
        
        return {
            id: `${routineId}-${nodeId}`,
            routineId,
            nodeId,
        };
    }

    private evaluateCondition(condition: string, context: Record<string, unknown>): boolean {
        // Create a safe evaluation context
        const safeContext = {
            ...context,
            // Add safe utility functions
            isEmpty: (val: unknown) => !val || (Array.isArray(val) && val.length === 0),
            isNumber: (val: unknown) => typeof val === "number",
            isString: (val: unknown) => typeof val === "string",
            isBoolean: (val: unknown) => typeof val === "boolean",
            isArray: Array.isArray,
        };

        try {
            // Use Function constructor for safer evaluation
            const keys = Object.keys(safeContext);
            const values = Object.values(safeContext);
            
            const func = new Function(...keys, `return ${condition}`);
            return func(...values);
        } catch (error) {
            this.logger.warn("[NativeNavigator] Condition evaluation failed", {
                condition,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    private traceBranch(startNodeId: string, def: NativeRoutineDefinition): Location[] {
        const branch: Location[] = [];
        const visited = new Set<string>();
        let currentNodeId = startNodeId;

        while (currentNodeId && !visited.has(currentNodeId)) {
            visited.add(currentNodeId);
            branch.push(this.createLocation(currentNodeId, def));

            // Find next node
            const outgoingEdges = def.edges.filter(e => e.from === currentNodeId);
            
            if (outgoingEdges.length === 0) {
                break; // End of branch
            }

            if (outgoingEdges.length > 1) {
                // Branch diverges - stop here
                break;
            }

            currentNodeId = outgoingEdges[0].to;
        }

        return branch;
    }
}

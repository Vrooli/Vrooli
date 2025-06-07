import { type Logger } from "winston";
import {
    type Location,
    type Navigator,
    type StepInfo,
    type OptimizationSuggestion,
} from "@vrooli/shared";

/**
 * Path analysis result
 */
export interface PathAnalysis {
    paths: Path[];
    criticalPath: Path;
    parallelizableGroups: StepGroup[];
    redundantSteps: string[];
    estimatedDuration: number;
}

/**
 * Execution path
 */
export interface Path {
    id: string;
    steps: string[];
    duration: number;
    probability: number;
    dependencies: string[][];
}

/**
 * Group of steps that can be parallelized
 */
export interface StepGroup {
    steps: string[];
    estimatedImprovement: number;
    constraints: string[];
}

/**
 * Path optimization options
 */
export interface OptimizationOptions {
    maxParallelism?: number;
    preserveOrder?: boolean;
    riskTolerance?: "low" | "medium" | "high";
}

/**
 * PathOptimizer - Analyzes and optimizes workflow execution paths
 * 
 * This component provides static analysis of workflow paths to identify
 * optimization opportunities before execution. It analyzes:
 * 
 * - Critical path identification
 * - Parallelization opportunities
 * - Step reordering possibilities
 * - Redundant step detection
 * - Conditional path optimization
 * 
 * The optimizer uses graph algorithms and heuristics to suggest
 * improvements that can significantly reduce execution time.
 */
export class PathOptimizer {
    private readonly logger: Logger;
    private readonly performanceCache: Map<string, number> = new Map();

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Analyzes paths in a routine
     */
    async analyzePaths(
        routineDefinition: unknown,
        navigator: Navigator,
        options: OptimizationOptions = {},
    ): Promise<PathAnalysis> {
        this.logger.debug("[PathOptimizer] Analyzing paths");

        try {
            // Get all possible paths
            const paths = await this.discoverPaths(routineDefinition, navigator);
            
            // Identify critical path
            const criticalPath = await this.findCriticalPath(paths);
            
            // Find parallelization opportunities
            const parallelizableGroups = await this.findParallelizableSteps(
                routineDefinition,
                navigator,
                options,
            );
            
            // Detect redundant steps
            const redundantSteps = await this.findRedundantSteps(paths);
            
            // Estimate total duration
            const estimatedDuration = await this.estimateDuration(
                criticalPath,
                parallelizableGroups,
            );

            const analysis: PathAnalysis = {
                paths,
                criticalPath,
                parallelizableGroups,
                redundantSteps,
                estimatedDuration,
            };

            this.logger.info("[PathOptimizer] Path analysis complete", {
                pathCount: paths.length,
                criticalPathLength: criticalPath.steps.length,
                parallelGroupCount: parallelizableGroups.length,
                redundantStepCount: redundantSteps.length,
            });

            return analysis;

        } catch (error) {
            this.logger.error("[PathOptimizer] Path analysis failed", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Suggests optimizations based on analysis
     */
    async suggestOptimizations(
        analysis: PathAnalysis,
        options: OptimizationOptions = {},
    ): Promise<OptimizationSuggestion[]> {
        const suggestions: OptimizationSuggestion[] = [];

        // Suggest parallelization
        for (const group of analysis.parallelizableGroups) {
            if (group.estimatedImprovement > 0.2) { // 20% improvement threshold
                suggestions.push({
                    type: "parallelize",
                    targetSteps: group.steps,
                    expectedImprovement: group.estimatedImprovement,
                    rationale: `These ${group.steps.length} steps have no dependencies and can run in parallel`,
                    risk: this.assessRisk(group, options.riskTolerance),
                });
            }
        }

        // Suggest removing redundant steps
        if (analysis.redundantSteps.length > 0) {
            suggestions.push({
                type: "skip",
                targetSteps: analysis.redundantSteps,
                expectedImprovement: analysis.redundantSteps.length / analysis.criticalPath.steps.length,
                rationale: "These steps produce outputs that are never used",
                risk: "low",
            });
        }

        // Suggest caching for frequently executed steps
        const frequentSteps = await this.findFrequentSteps(analysis.paths);
        for (const stepId of frequentSteps) {
            suggestions.push({
                type: "cache",
                targetSteps: [stepId],
                expectedImprovement: 0.7, // 70% improvement on cache hits
                rationale: "This step is executed in multiple paths and could benefit from caching",
                risk: "low",
            });
        }

        // Sort by expected improvement
        suggestions.sort((a, b) => b.expectedImprovement - a.expectedImprovement);

        return suggestions;
    }

    /**
     * Discovers all possible execution paths
     */
    private async discoverPaths(
        routineDefinition: unknown,
        navigator: Navigator,
    ): Promise<Path[]> {
        const paths: Path[] = [];
        const visited = new Set<string>();
        
        const startLocation = navigator.getStartLocation(routineDefinition);
        
        // DFS to find all paths
        await this.explorePath(
            routineDefinition,
            navigator,
            startLocation,
            [],
            visited,
            paths,
            1.0, // Initial probability
        );

        return paths;
    }

    /**
     * Explores a path using DFS
     */
    private async explorePath(
        routineDefinition: unknown,
        navigator: Navigator,
        location: Location,
        currentPath: string[],
        visited: Set<string>,
        paths: Path[],
        probability: number,
    ): Promise<void> {
        // Avoid cycles
        if (visited.has(location.nodeId)) {
            return;
        }

        visited.add(location.nodeId);
        currentPath.push(location.nodeId);

        // Check if end location
        if (navigator.isEndLocation(location)) {
            // Create path
            const path: Path = {
                id: `path-${paths.length + 1}`,
                steps: [...currentPath],
                duration: await this.estimatePathDuration(currentPath),
                probability,
                dependencies: await this.extractDependencies(currentPath, navigator, location),
            };
            paths.push(path);
        } else {
            // Get next locations
            const nextLocations = navigator.getNextLocations(location, {});
            
            // Calculate probability for each branch
            const branchProbability = probability / Math.max(nextLocations.length, 1);
            
            // Explore each branch
            for (const nextLocation of nextLocations) {
                await this.explorePath(
                    routineDefinition,
                    navigator,
                    nextLocation,
                    currentPath,
                    new Set(visited), // Clone visited set for each branch
                    paths,
                    branchProbability,
                );
            }
        }

        // Backtrack
        currentPath.pop();
        visited.delete(location.nodeId);
    }

    /**
     * Finds the critical path (longest duration)
     */
    private async findCriticalPath(paths: Path[]): Promise<Path> {
        if (paths.length === 0) {
            throw new Error("No paths found");
        }

        let criticalPath = paths[0];
        let maxDuration = 0;

        for (const path of paths) {
            const weightedDuration = path.duration * path.probability;
            if (weightedDuration > maxDuration) {
                maxDuration = weightedDuration;
                criticalPath = path;
            }
        }

        return criticalPath;
    }

    /**
     * Finds steps that can be parallelized
     */
    private async findParallelizableSteps(
        routineDefinition: unknown,
        navigator: Navigator,
        options: OptimizationOptions,
    ): Promise<StepGroup[]> {
        const groups: StepGroup[] = [];
        const startLocation = navigator.getStartLocation(routineDefinition);
        
        // Find parallel branches
        const parallelBranches = navigator.getParallelBranches(startLocation);
        
        for (const branches of parallelBranches) {
            if (branches.length > 1) {
                const steps = branches.map(branch => branch[0]?.nodeId).filter(Boolean);
                
                if (steps.length > 1) {
                    const group: StepGroup = {
                        steps,
                        estimatedImprovement: this.calculateParallelImprovement(steps.length),
                        constraints: [],
                    };
                    
                    // Add constraints
                    if (options.maxParallelism && steps.length > options.maxParallelism) {
                        group.constraints.push(`Limited to ${options.maxParallelism} parallel executions`);
                    }
                    
                    groups.push(group);
                }
            }
        }

        return groups;
    }

    /**
     * Finds redundant steps
     */
    private async findRedundantSteps(paths: Path[]): Promise<string[]> {
        const stepOutputUsage = new Map<string, Set<string>>();
        const allSteps = new Set<string>();

        // Collect all steps and their output usage
        for (const path of paths) {
            for (let i = 0; i < path.steps.length; i++) {
                const step = path.steps[i];
                allSteps.add(step);
                
                // Check if this step's outputs are used by subsequent steps
                for (let j = i + 1; j < path.steps.length; j++) {
                    const consumer = path.steps[j];
                    if (!stepOutputUsage.has(step)) {
                        stepOutputUsage.set(step, new Set());
                    }
                    stepOutputUsage.get(step)!.add(consumer);
                }
            }
        }

        // Find steps whose outputs are never used
        const redundant: string[] = [];
        for (const step of allSteps) {
            const consumers = stepOutputUsage.get(step);
            if (!consumers || consumers.size === 0) {
                // Check if it's not a final step
                const isFinalStep = paths.some(p => p.steps[p.steps.length - 1] === step);
                if (!isFinalStep) {
                    redundant.push(step);
                }
            }
        }

        return redundant;
    }

    /**
     * Finds frequently executed steps across paths
     */
    private async findFrequentSteps(paths: Path[]): Promise<string[]> {
        const stepFrequency = new Map<string, number>();
        
        for (const path of paths) {
            for (const step of path.steps) {
                const count = stepFrequency.get(step) || 0;
                stepFrequency.set(step, count + path.probability);
            }
        }

        // Find steps executed in >50% of paths
        const frequent: string[] = [];
        for (const [step, frequency] of stepFrequency) {
            if (frequency > 0.5) {
                frequent.push(step);
            }
        }

        return frequent;
    }

    /**
     * Estimates path duration
     */
    private async estimatePathDuration(steps: string[]): Promise<number> {
        let totalDuration = 0;
        
        for (const step of steps) {
            // Use cached performance data if available
            const cachedDuration = this.performanceCache.get(step);
            if (cachedDuration) {
                totalDuration += cachedDuration;
            } else {
                // Default estimate based on step type
                totalDuration += 1000; // 1 second default
            }
        }

        return totalDuration;
    }

    /**
     * Estimates total duration with optimizations
     */
    private async estimateDuration(
        criticalPath: Path,
        parallelGroups: StepGroup[],
    ): Promise<number> {
        let duration = criticalPath.duration;
        
        // Apply parallelization improvements
        for (const group of parallelGroups) {
            const improvement = duration * group.estimatedImprovement;
            duration -= improvement;
        }

        return Math.max(duration, 0);
    }

    /**
     * Extracts dependencies for a path
     */
    private async extractDependencies(
        path: string[],
        navigator: Navigator,
        endLocation: Location,
    ): Promise<string[][]> {
        const dependencies: string[][] = [];
        
        // For each step, get its dependencies
        for (const stepId of path) {
            const location: Location = {
                ...endLocation,
                nodeId: stepId,
            };
            const deps = navigator.getDependencies(location);
            if (deps.length > 0) {
                dependencies.push(deps);
            }
        }

        return dependencies;
    }

    /**
     * Calculates improvement from parallelization
     */
    private calculateParallelImprovement(branchCount: number): number {
        // Amdahl's Law approximation
        // Assumes 80% of work can be parallelized
        const parallelFraction = 0.8;
        const serialFraction = 1 - parallelFraction;
        
        const speedup = 1 / (serialFraction + parallelFraction / branchCount);
        const improvement = (speedup - 1) / speedup;
        
        return Math.min(improvement, 0.9); // Cap at 90% improvement
    }

    /**
     * Assesses risk level
     */
    private assessRisk(
        group: StepGroup,
        tolerance?: "low" | "medium" | "high",
    ): "low" | "medium" | "high" {
        // Assess based on group size and constraints
        if (group.steps.length > 5 || group.constraints.length > 2) {
            return "high";
        }
        
        if (group.steps.length > 3 || group.constraints.length > 0) {
            return "medium";
        }
        
        return "low";
    }

    /**
     * Updates performance cache
     */
    async updatePerformanceData(
        stepId: string,
        averageDuration: number,
    ): Promise<void> {
        this.performanceCache.set(stepId, averageDuration);
    }
}

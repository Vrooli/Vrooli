import { RunProject, RunProjectCreateInput, RunProjectUpdateInput, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput } from "../api/types.js";
import { PassableLogger } from "../consts/commonTypes.js";
import { RunProjectShape, RunProjectStepShape, RunRoutineShape, RunRoutineStepShape, shapeRunProject, shapeRunRoutine } from "../shape/models/models.js";
import { RunProgressConfig } from "./configs/run.js";
import { FINALIZE_RUN_POLL_INTERVAL_MS, FINALIZE_RUN_TIMEOUT_MS, STORE_RUN_PROGRESS_DEBOUNCE_MS } from "./consts.js";
import { RunIdentifier, RunMetrics, RunProgress, RunProgressStep, RunTriggeredBy } from "./types.js";

/**
 * Handles saving and loading run progress. This includes:
 * - Debouncing multiple `saveProgress` calls to avoid redundant DB writes
 * - Caching the current run in memory to avoid re-fetching
 * - Converting between a shape useful to the state machine (RunProgress) and 
 *   the shape needed to store in the database (RunProjectShape or RunRoutineShape)
 * 
 * NOTE: This class assumes that only one run is being worked on at a time.
 * 
 * When a run is completed, you must call `finalizeSave(true)` to ensure the final 
 * state is persisted to the database and all caches are cleared.
 */
export abstract class RunPersistence {
    /**
     * The in-memory run currently loaded. If you call loadProgress with the
     * same run ID, we just return this instead of re-fetching.
     */
    private _currentRun: RunProgress | null = null;

    /** 
     * The last shape we persisted to the DB (or null if we haven't yet).
     * This helps us know if we need to 'create' or 'update', and to generate diffs.
     */
    private _lastStoredShape: RunProjectShape | RunRoutineShape | null = null;

    /**
     * We only keep a single "pending" run in memory. If `saveProgress` is called repeatedly,
     * each call overwrites `_pendingRun` so only the final state is persisted.
     */
    private _pendingRun: RunProgress | null = null;

    /** True if a store operation is currently in progress (we do not run multiple in parallel). */
    private _storeInProgress = false;

    /** Debounce timer ID for scheduling the next flush of `_pendingRun`. */
    private _debounceTimer: NodeJS.Timeout | null = null;

    /** How long to wait after a `saveProgress` call before flushing to DB. */
    private DEBOUNCE_DELAY_MS = STORE_RUN_PROGRESS_DEBOUNCE_MS;

    /**
     * -------------------------------------------------------------------------
     * ABSTRACT METHODS: to be implemented by state machine's environment
     * -------------------------------------------------------------------------
     */

    /**
     * Low-level database or API call that CREATEs a new RunProject
     * and returns the *final* shape (including any DB-assigned IDs).
     * 
     * @param input The input to create the run project
     * @returns The final shape of the created run project, or null if it failed
     */
    protected abstract createRunProject(input: RunProjectCreateInput): Promise<RunProject | null>;

    /**
     * Low-level database or API call that CREATEs a new RunRoutine
     * and returns the *final* shape (including any DB-assigned IDs).
     * 
     * @param input The input to create the run routine
     * @returns The final shape of the created run routine, or null if it failed
     */
    protected abstract createRunRoutine(input: RunRoutineCreateInput): Promise<RunRoutine | null>;

    /**
     * Low-level database or API call that UPDATEs an existing RunProject.
     * Returns the *final* shape after the update.
     * 
     * @param input The input to update the run project
     * @returns The final shape of the updated run project, or null if it failed
     */
    protected abstract updateRunProject(input: RunProjectUpdateInput): Promise<RunProject | null>;

    /**
     * Low-level database or API call that UPDATEs an existing RunRoutine.
     * Returns the *final* shape after the update.
     * 
     * @param input The input to update the run routine
     * @returns The final shape of the updated run routine, or null if it failed
     */
    protected abstract updateRunRoutine(input: RunRoutineUpdateInput): Promise<RunRoutine | null>;

    /**
     * Fetch the run progress from the database or call an API to do so.
     * 
     * @param run Information required to fetch the run
     * @returns The fetched run data, or null if not found or an error occurred
     */
    protected abstract fetchRunProgress(run: RunIdentifier): Promise<RunProject | RunRoutine | null>;

    /**
     * -------------------------------------------------------------------------
     * PUBLIC METHODS
     * -------------------------------------------------------------------------
     */

    /**
     * Loads run progress. If the requested run ID is the same as our currently
     * loaded `_currentRun`, just return that. Otherwise, fetch from DB/API.
     * 
     * @param run The run to load
     * @param userData Session data for the user running the routine
     * @param logger The logger to use for any errors
     */
    public async loadProgress(
        run: RunIdentifier,
        userData: RunTriggeredBy,
        logger: PassableLogger,
    ): Promise<RunProgress | null> {
        // If we already have a run in memory with the same ID, return it
        if (this._currentRun && this._currentRun.runId === run.runId) {
            return this._currentRun;
        }

        // Otherwise, fetch fresh
        const loaded = await this.fetchRunProgress(run);
        if (loaded) {
            const shaped = loaded.__typename === "RunProject"
                ? this.fromFetchedProject(loaded, userData, logger)
                : this.fromFetchedRoutine(loaded, userData, logger);
            this._currentRun = shaped;
        }
        return this._currentRun;
    }

    /**
     * Schedules a save of this run progress. We'll debounce actual writes so that
     * repeated calls within DEBOUNCE_DELAY only cause a single DB operation.
     */
    public saveProgress(run: RunProgress): void {
        // Overwrite what we have in memory
        this._currentRun = run;

        // Store as "pending". If multiple calls happen quickly, only the final one is used.
        this._pendingRun = run;

        // If we are in the middle of an ongoing store, do NOT start a new timer.
        // We'll schedule after the ongoing store finishes.
        if (this._storeInProgress) {
            return;
        }

        // Otherwise, start (or restart) the debounce timer
        this.scheduleDebounce();
    }

    /**
     * Wait until all pending writes have completed or a timeout occurs.
     * 
     * @param flushCache If true, also flushes any in-memory caches.
     * @param timeoutMs How long to wait before giving up.
     * @returns True if all writes completed, false if we timed out.
     */
    public async finalizeSave(flushCache = true, timeoutMs = FINALIZE_RUN_TIMEOUT_MS): Promise<boolean> {
        const startTime = Date.now();

        // We'll poll for the condition: "no store in progress, no pending run, no timer"
        // If we reach that condition, we return true. If we exceed `timeoutMs`, return false.
        return new Promise<boolean>((resolve) => {
            const check = () => {
                const now = Date.now();
                if (
                    !this._storeInProgress && // not currently storing
                    !this._pendingRun &&      // no new data pending
                    !this._debounceTimer      // no scheduled flush
                ) {
                    // Reset everything if we're flushing the cache
                    if (flushCache) {
                        this._pendingRun = null;
                        this._lastStoredShape = null;
                        this._currentRun = null;
                        this._storeInProgress = false;
                        if (this._debounceTimer) {
                            clearTimeout(this._debounceTimer);
                            this._debounceTimer = null;
                        }
                    }

                    resolve(true);
                } else if (now - startTime > timeoutMs) {
                    resolve(false);
                } else {
                    // Check again in a short interval
                    setTimeout(check, FINALIZE_RUN_POLL_INTERVAL_MS);
                }
            };

            check();
        });
    }

    /**
     * -------------------------------------------------------------------------
     * INTERNAL METHODS: Debounce & Store
     * -------------------------------------------------------------------------
     */

    /**
     * Schedules a flush after DEBOUNCE_DELAY. Clears any previous timer.
     */
    private scheduleDebounce(): void {
        if (this._debounceTimer) {
            clearTimeout(this._debounceTimer);
        }
        this._debounceTimer = setTimeout(() => this.flushPending(), this.DEBOUNCE_DELAY_MS);
    }

    /**
     * Called after the debounce interval. If no store is in progress, flush `_pendingRun`.
     */
    private async flushPending(): Promise<void> {
        this._debounceTimer = null;
        // If there's already a store in progress, do nothing:
        if (this._storeInProgress) return;

        // If there's nothing pending, do nothing
        if (!this._pendingRun) return;

        // Perform the store with the final pending run
        await this.performStore(this._pendingRun);
    }

    /**
     * Helper to call the correct shape conversion
     * 
     * @param run The run to store
     * @returns The shape to store
     */
    private toShape(run: RunProgress): RunProjectShape | RunRoutineShape {
        return run.type === "RunProject"
            ? this.toProjectShape(run)
            : this.toRoutineShape(run);
    }

    /**
     * Helper to call the correct store method for creating a run.
     * 
     * @param runShape The shape to store
     * @returns The result of the store operation, or null if it failed
     */
    private async createRun(runShape: RunProjectShape | RunRoutineShape): Promise<RunProject | RunRoutine | null> {
        if (runShape.__typename === "RunProject") {
            const input = shapeRunProject.create(runShape as RunProjectShape);
            return await this.createRunProject(input);
        }
        const input = shapeRunRoutine.create(runShape as RunRoutineShape);
        return await this.createRunRoutine(input);
    }

    /**
     * Helper to call the correct store method for updating a run.
     * 
     * @param runShape The shape to store
     * @returns The result of the store operation, or null if no shape was stored 
     * (due to no changes or other reasons)
     */
    private async updateRun(runShape: RunProjectShape | RunRoutineShape): Promise<RunProject | RunRoutine | null> {
        const originalShape = this._lastStoredShape;
        if (!originalShape) {
            return null;
        }
        if (runShape.__typename === "RunProject") {
            const input = shapeRunProject.update(originalShape as RunProjectShape, runShape as RunProjectShape);
            if (!input) {
                return null;
            }
            return await this.updateRunProject(input);
        }
        const input = shapeRunRoutine.update(originalShape as RunRoutineShape, runShape as RunRoutineShape);
        if (!input) {
            return null;
        }
        return await this.updateRunRoutine(input);
    }

    /**
     * Actually store the data in `_pendingRun`.
     * 1) Mark `_storeInProgress = true`
     * 2) Convert to shape
     * 3) Create vs. Update
     * 4) Update `_lastStoredShape`, `_pendingRun`, etc.
     * 5) If `_pendingRun` changed *during* the store, schedule a new flush.
     */
    private async performStore(run: RunProgress): Promise<void> {
        this._storeInProgress = true;

        try {
            const shapeToStore = this.toShape(run);

            const isCreate =
                !this._lastStoredShape ||  // haven't stored anything yet
                this._lastStoredShape.__typename !== shapeToStore.__typename || // changed type?
                !run.runId;                  // no real ID means brand-new

            const storeResult = isCreate
                ? await this.createRun(shapeToStore)
                : await this.updateRun(shapeToStore);

            if (!storeResult) {
                // If the update didn't actually change anything, we don't need to do anything else.
                return;
            }

            // Update the run ID (in case the DB assigned/returned a new/different ID)
            run.runId = storeResult.id;

            // Remember the shape we just saved
            this._lastStoredShape = shapeToStore;
        } catch (err) {
            console.error("Error performing store:", err);
            // Depending on your needs, you might re-throw or handle differently
        } finally {
            this._storeInProgress = false;

            // If _pendingRun has changed since we started, we should do another debounce
            // so we can coalesce any new calls again.
            if (this._pendingRun !== run && this._pendingRun != null) {
                this.scheduleDebounce();
            }
        }
    }

    /**
     * ----------------------------------------------------------------------------
     * TRANSFORMS: from RunProgress -> RunProjectShape or RunRoutineShape
     * ----------------------------------------------------------------------------
     */

    /**
     * Sums the context switches from all steps in the run.
     * @param run The run to calculate context switches for
     * @returns The total context switches
     */
    private calculateTotalContextSwitches(run: RunProgress): number {
        // Sum up all context switches from each step
        return run.steps.reduce((sum, s) => sum + (s.contextSwitches || 0), 0);
    }

    /**
     * Sums the complexity of all steps in the run.
     * @param run The run to calculate complexity for
     * @returns The total complexity
     */
    private calculateTotalCompletedComplexity(run: RunProgress): number {
        // Sum up the complexity of each step
        return run.steps.reduce(function complexityAccumulator(sum, s) {
            return sum + (s.complexity || 0);
        }, 0);
    }

    /**
     * Extracts the team from the run, if it is being run by a team.
     * Otherwise, returns null.
     * @param run The run to extract the team from
     * @returns The team, or null if the run is not owned by a team
     */
    private getRunTeam(run: RunProgress) {
        return run.owner.__typename === "Team"
            ? { __typename: "Team" as const, id: run.owner.id, __connect: true }
            : null;
    }

    /**
     * Calculates the time elapsed for a run step, in milliseconds.
     * @param step The step to calculate the time elapsed for
     * @returns The time elapsed for the step, in milliseconds
     */
    private calculateStepTimeElapsed(step: RunProgressStep): number {
        // Can only calculate if there is a start and end time
        if (!step.startedAt || !step.completedAt) {
            return 0;
        }
        // We're using Date objects, so we can just subtract them to get the difference in milliseconds
        return step.completedAt.getTime() - step.startedAt.getTime();
    }

    /**
     * Shapes a RunProgressStep into a RunProjectStepShape.
     * @param step The run step to shape
     * @param index The index of the step in the run
     * @param runId The ID of the run
     * @returns The shaped step
     */
    private toProjectStepShape(
        step: RunProgressStep,
        index: number,
        runId: string,
    ): RunProjectStepShape {
        const { completedAt, complexity, contextSwitches, id, name, startedAt, status } = step;
        const directory = step.subroutineId ?
            { id: step.subroutineId, __typename: "ProjectVersionDirectory" as const }
            : null;
        const directoryInId = step.objectId;
        const runProject = { id: runId, __typename: "RunProject" as const };
        const timeElapsed = this.calculateStepTimeElapsed(step);

        return {
            __typename: "RunProjectStep" as const,
            id,
            completedAt,
            complexity,
            contextSwitches,
            directory,
            directoryInId,
            name,
            order: index,
            startedAt,
            status,
            timeElapsed,
            runProject,
        };
    }

    /**
     * Shapes a RunProgressStep into a RunRoutineStepShape.
     * @param step The run step to shape
     * @param index The index of the step in the run
     * @param runId The ID of the run
     * @returns The shaped step
     */
    private toRoutineStepShape(
        step: RunProgressStep,
        index: number,
        runId: string,
    ): RunRoutineStepShape {
        const { completedAt, complexity, contextSwitches, id, name, startedAt, status } = step;
        const nodeId = step.locationId;
        const runRoutine = { id: runId, __typename: "RunRoutine" as const };
        const subroutine = step.subroutineId ?
            { id: step.subroutineId, __typename: "RoutineVersion" as const }
            : null;
        const subroutineInId = step.objectId;
        const timeElapsed = this.calculateStepTimeElapsed(step);

        return {
            __typename: "RunRoutineStep" as const,
            id,
            completedAt,
            complexity,
            contextSwitches,
            name,
            nodeId,
            order: index,
            startedAt,
            status,
            timeElapsed,
            runRoutine,
            subroutine,
            subroutineInId,
        };
    }

    private toProjectShape(run: RunProgress): RunProjectShape {
        // Build the config object to store in the "data" field
        const config = new RunProgressConfig(run);
        const dataString = config.serialize("json");

        const contextSwitches = this.calculateTotalContextSwitches(run);
        const completedComplexity = this.calculateTotalCompletedComplexity(run);
        const timeElapsed = run.metrics.timeElapsed || 0;

        const projectVersion = run.runOnObjectId
            ? { id: run.runOnObjectId, __typename: "ProjectVersion" as const }
            : null;
        const schedule = run.schedule ?? null;
        const steps = run.steps.map((step, index) => this.toProjectStepShape(step, index, run.runId));
        const team = this.getRunTeam(run);

        const shape: RunProjectShape = {
            __typename: "RunProject" as const,
            id: run.runId,
            isPrivate: run.config.isPrivate,
            completedComplexity,
            contextSwitches,
            data: dataString,
            name: run.name,
            status: run.status,
            timeElapsed,
            steps,
            schedule,
            projectVersion,
            team,
        };
        return shape;
    }

    private toRoutineShape(run: RunProgress): RunRoutineShape {
        // Build the config object to store in the "data" field
        const config = new RunProgressConfig(run);
        const dataString = config.serialize("json");

        const contextSwitches = this.calculateTotalContextSwitches(run);
        const completedComplexity = this.calculateTotalCompletedComplexity(run);
        const timeElapsed = run.metrics.timeElapsed || 0;

        const routineVersion = run.runOnObjectId
            ? { id: run.runOnObjectId, __typename: "RoutineVersion" as const }
            : null;
        const schedule = run.schedule ?? null;
        const steps = run.steps.map((step, index) => this.toRoutineStepShape(step, index, run.runId));
        const team = this.getRunTeam(run);

        //TODO
        // For “io,” we might merge all inputs/outputs from each subcontext.
        // This is one common approach; adapt as needed.
        const io: any[] = [];// IOEntry[] = [];
        for (const [subId, subCtx] of Object.entries(run.subcontexts)) {
            // Combine inputs and outputs from each subcontext
            subCtx.allInputsList?.forEach((entry) => {
                io.push({ key: `${subId}.input.${entry.key}`, value: entry.value });
            });
            subCtx.allOutputsList?.forEach((entry) => {
                io.push({ key: `${subId}.output.${entry.key}`, value: entry.value });
            });
        }

        const shape: RunRoutineShape = {
            __typename: "RunRoutine" as const,
            id: run.runId,
            isPrivate: run.config.isPrivate,
            completedComplexity,
            contextSwitches,
            data: dataString,
            name: run.name,
            status: run.status,
            timeElapsed,
            steps,
            io,
            schedule,
            routineVersion,
            team,
        };
        return shape;
    }

    /**
     * Converts a stored RunProject or RunRoutine into a RunMetrics object.
     * 
     * @param storedData The stored data to convert
     * @param creditsSpent The credits spent on the run so far
     * @returns The metrics object
     */
    private storedDataToMetrics(storedData: RunProject | RunRoutine, creditsSpent: string): RunMetrics {
        let complexityTotal = 0;
        if (storedData.__typename === "RunRoutine") {
            complexityTotal = (storedData as RunRoutine).routineVersion?.complexity || 0;
        } else if (storedData.__typename === "RunProject") {
            complexityTotal = (storedData as RunProject).projectVersion?.complexity || 0;
        }

        const metrics: RunMetrics = {
            complexityCompleted: Math.max(storedData.completedComplexity || 0, 0),
            complexityTotal: Math.max(complexityTotal, 0),
            timeElapsed: Math.max(storedData.timeElapsed || 0, 0),
            stepsRun: Math.max(storedData.steps?.length || 0, 0),
            creditsSpent,
        };
        return metrics;
    }

    /**
     * Converts a stored RunProject or RunRoutine into an owner object.
     * 
     * @param storedData The stored data to convert
     * @param userData Session data for the user running the routine
     * @returns The owner object
     */
    private storedDataToOwner(
        storedData: RunProject | RunRoutine,
        userData: RunTriggeredBy,
    ): RunProgress["owner"] {
        if (storedData.team) {
            return { id: storedData.team.id, __typename: "Team" as const };
        }
        return { id: userData.id, __typename: "User" as const };
    }

    /**
     * Shapes a RunProjectStepShape into a RunProgressStep.
     * @param step The run step shape to convert
     * @returns The reconstructed run progress step
     */
    private fromProjectStepShape(step: RunProjectStepShape): RunProgressStep {
        return {
            __typename: "ProjectVersion" as const,
            completedAt: step.completedAt || undefined,
            complexity: step.complexity || 0,
            contextSwitches: step.contextSwitches || 0,
            id: step.id,
            locationId: "", // Not used for project steps
            name: step.name,
            objectId: step.directoryInId, // The ID of the project version the directory is a part of. May be different than the project version being run if we step into a nested project
            startedAt: step.startedAt,
            status: step.status,
            subroutineId: step.directory?.id || null,
        };
    }

    /**
     * Shapes the steps in a stored RunProject into RunProgressSteps, 
     * and sorts them by order.
     * 
     * @param storedData The stored project to convert
     * @returns The reconstructed run progress steps
     */
    private fromProjectSteps(storedData: RunProject): RunProgressStep[] {
        return storedData.steps
            .sort((a, b) => a.order - b.order)
            .map(this.fromProjectStepShape);
    }

    /**
     * Shapes a RunRoutineStepShape into a RunProgressStep.
     * @param step The run step shape to convert
     * @returns The reconstructed run progress step
     */
    private fromRoutineStepShape(step: RunRoutineStepShape): RunProgressStep {
        return {
            __typename: "RoutineVersion" as const,
            completedAt: step.completedAt || undefined,
            complexity: step.complexity || 0,
            contextSwitches: step.contextSwitches || 0,
            id: step.id,
            locationId: step.nodeId,
            name: step.name,
            objectId: step.subroutineInId, // The ID of the routine version the node is a part of. May be different than the routine version being run if we step into a nested routine
            startedAt: step.startedAt,
            status: step.status,
            subroutineId: step.subroutine?.id || null,
        };
    }

    /**
     * Shapes the steps in a stored RunRoutine into RunProgressSteps,
     * and sorts them by order.
     * 
     * @param storedData The stored routine to convert
     * @returns The reconstructed run progress steps
     */
    private fromRoutineSteps(storedData: RunRoutine): RunProgressStep[] {
        return storedData.steps
            .sort((a, b) => a.order - b.order)
            .map(this.fromRoutineStepShape);
    }

    /**
     * Converts a stored RunProject into a RunProgress object.
     * 
     * @param storedData The stored project to convert
     * @param userData Session data for the user running the routine
     * @param logger The logger to use for any errors
     */
    private fromFetchedProject(
        storedData: RunProject,
        userData: RunTriggeredBy,
        logger: PassableLogger,
    ): RunProgress {
        const { name, id: runId, status } = storedData;
        const { __version, branches, config, decisions, metrics: partialMetrics, subcontexts } = storedData.data
            ? RunProgressConfig.deserialize(storedData, logger)
            : RunProgressConfig.default();
        const metrics = this.storedDataToMetrics(storedData, partialMetrics.creditsSpent);
        const owner = this.storedDataToOwner(storedData, userData);
        const schedule = storedData.schedule ?? null;
        const steps = this.fromProjectSteps(storedData);

        const progress: RunProgress = {
            __version,
            branches,
            config: { ...config, isPrivate: storedData.isPrivate },
            decisions,
            name,
            owner,
            runId,
            runOnObjectId: storedData.projectVersion?.id || null,
            schedule,
            steps,
            metrics,
            status,
            subcontexts,
            type: "RunProject" as const,
        };
        return progress;
    }

    /**
     * Converts a stored RunRoutine into a RunProgress object.
     * 
     * @param storedData The stored data to convert
     * @param userData Session data for the user running the routine
     * @param logger The logger to use for any errors
     */
    private fromFetchedRoutine(
        storedData: RunRoutine,
        userData: RunTriggeredBy,
        logger: PassableLogger,
    ): RunProgress {
        const { name, id: runId, status } = storedData;
        const { __version, branches, config, decisions, metrics: partialMetrics, subcontexts } = storedData.data
            ? RunProgressConfig.deserialize(storedData, logger)
            : RunProgressConfig.default();
        const metrics = this.storedDataToMetrics(storedData, partialMetrics.creditsSpent);
        const owner = this.storedDataToOwner(storedData, userData);
        const schedule = storedData.schedule ?? null;
        const steps = this.fromRoutineSteps(storedData);

        const progress: RunProgress = {
            __version,
            branches,
            config: { ...config, isPrivate: storedData.isPrivate },
            decisions,
            name,
            owner,
            runId,
            runOnObjectId: storedData.routineVersion?.id || null,
            schedule,
            steps,
            metrics,
            status,
            subcontexts,
            type: "RunRoutine" as const,
        };
        return progress;
    }

}

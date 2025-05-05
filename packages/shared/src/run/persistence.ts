import { Run, RunCreateInput, RunUpdateInput } from "../api/types.js";
import { PassableLogger } from "../consts/commonTypes.js";
import { RunProgressConfig } from "../shape/configs/run.js";
import { RunShape, RunStepShape, shapeRun } from "../shape/models/models.js";
import { FINALIZE_RUN_POLL_INTERVAL_MS, FINALIZE_RUN_TIMEOUT_MS, STORE_RUN_PROGRESS_DEBOUNCE_MS } from "./consts.js";
import { RunIdentifier, RunMetrics, RunProgress, RunProgressStep, RunTriggeredBy } from "./types.js";

/**
 * Handles saving and loading run progress. This includes:
 * - Debouncing multiple `saveProgress` calls to avoid redundant DB writes
 * - Caching the current run in memory to avoid re-fetching
 * - Converting between a shape useful to the state machine (RunProgress) and 
 *   the shape needed to store in the database (RunShape)
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
    private _lastStoredShape: RunShape | null = null;

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
     * Low-level database or API call that CREATEs a new Run
     * and returns the *final* shape (including any DB-assigned IDs).
     * 
     * @param input The input to create the run
     * @returns The final shape of the created run, or null if it failed
     */
    protected abstract postRun(input: RunCreateInput): Promise<Run | null>;

    /**
     * Low-level database or API call that UPDATEs an existing Run.
     * Returns the *final* shape after the update.
     * 
     * @param input The input to update the run
     * @returns The final shape of the updated run, or null if it failed
     */
    protected abstract putRun(input: RunUpdateInput): Promise<Run | null>;

    /**
     * Fetch the run progress from the database or call an API to do so.
     * 
     * @param run Information required to fetch the run
     * @returns The fetched run data, or null if not found or an error occurred
     */
    protected abstract fetchRun(run: RunIdentifier): Promise<Run | null>;

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
     * @param throwIfNotLoaded When true, throws an error if the run isn't loaded.
     * @returns The loaded run progress, or null if it isn't loaded and `throwIfNotLoaded` is false.
     * @throws If the run isn't loaded and `throwIfNotLoaded` is true.
     */
    public async loadProgress<ThrowIfNotLoaded extends boolean = true>(
        run: RunIdentifier,
        userData: RunTriggeredBy,
        logger: PassableLogger,
        throwIfNotLoaded: ThrowIfNotLoaded = true as ThrowIfNotLoaded,
    ): Promise<ThrowIfNotLoaded extends true ? RunProgress : RunProgress | null> {
        // If we already have a run in memory with the same ID, return it
        if (this._currentRun && this._currentRun.runId === run.runId) {
            return this._currentRun;
        }

        // Otherwise, fetch fresh
        const loaded = await this.fetchRun(run);
        if (loaded) {
            const shaped = this.fromFetchedRun(loaded, userData, logger);
            this._currentRun = shaped;
        }
        if (throwIfNotLoaded && !this._currentRun) {
            throw new Error(`Run ID ${run.runId} not found in database`);
        }
        return this._currentRun as ThrowIfNotLoaded extends true ? RunProgress : RunProgress | null;
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
     * Helper for shaping and creating a run.
     * 
     * @param runShape The shape to store
     * @returns The result of the store operation, or null if it failed
     */
    private async createRun(runShape: RunShape): Promise<Run | null> {
        const input = shapeRun.create(runShape);
        return await this.postRun(input);
    }

    /**
     * Helper for shaping and updating a run.
     * 
     * @param runShape The shape to store
     * @returns The result of the store operation, or null if no shape was stored 
     * (due to no changes or other reasons)
     */
    private async updateRun(runShape: RunShape): Promise<Run | null> {
        const originalShape = this._lastStoredShape;
        if (!originalShape) {
            return null;
        }
        const input = shapeRun.update(originalShape, runShape);
        if (!input) {
            return null;
        }
        return await this.putRun(input);
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
            const shapeToStore = this.toRunShape(run);

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
     * Shapes a RunProgressStep into a RunStepShape.
     * @param step The run step to shape
     * @param index The index of the step in the run
     * @param runId The ID of the run
     * @returns The shaped step
     */
    private toRunStepShape(
        step: RunProgressStep,
        index: number,
        runId: string,
    ): RunStepShape {
        const { completedAt, complexity, contextSwitches, id, name, startedAt, status } = step;
        const nodeId = step.locationId;
        const run = { id: runId, __typename: "Run" as const };
        const resourceVersion = step.subroutineId ?
            { id: step.subroutineId, __typename: "ResourceVersion" as const }
            : null;
        const resourceInId = step.objectId;
        const timeElapsed = this.calculateStepTimeElapsed(step);

        return {
            __typename: "RunStep" as const,
            id,
            completedAt,
            complexity,
            contextSwitches,
            name,
            nodeId,
            order: index,
            run,
            startedAt,
            status,
            resourceInId,
            resourceVersion,
            timeElapsed,
        };
    }

    /**
     * Converts run progress into a shape to store in the database.
     * 
     * @param run The run to store
     * @returns The shape to store
     */
    private toRunShape(run: RunProgress): RunShape {
        // Build the config object to store in the "data" field
        const config = new RunProgressConfig(run);
        const dataString = config.serialize("json");

        const contextSwitches = this.calculateTotalContextSwitches(run);
        const completedComplexity = this.calculateTotalCompletedComplexity(run);
        const timeElapsed = run.metrics.timeElapsed || 0;

        const resourceVersion = run.runOnObjectId
            ? { id: run.runOnObjectId, __typename: "ResourceVersion" as const }
            : null;
        const schedule = run.schedule ?? null;
        const steps = run.steps.map((step, index) => this.toRunStepShape(step, index, run.runId));
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

        const shape: RunShape = {
            __typename: "Run" as const,
            id: run.runId,
            completedComplexity,
            contextSwitches,
            data: dataString,
            io,
            isPrivate: run.config.isPrivate,
            name: run.name,
            resourceVersion,
            schedule,
            startedAt: run.metrics.startedAt,
            status: run.status,
            steps,
            team,
            timeElapsed,
        };
        return shape;
    }

    /**
     * Converts a stored Run into a RunMetrics object.
     * 
     * @param storedData The stored data to convert
     * @param creditsSpent The credits spent on the run so far
     * @returns The metrics object
     */
    private storedDataToMetrics(storedData: Run, creditsSpent: string): RunMetrics {
        const complexityTotal = storedData.resourceVersion?.complexity || 0;

        const metrics: RunMetrics = {
            complexityCompleted: Math.max(storedData.completedComplexity || 0, 0),
            complexityTotal: Math.max(complexityTotal, 0),
            creditsSpent,
            startedAt: storedData.startedAt,
            stepsRun: Math.max(storedData.steps?.length || 0, 0),
            timeElapsed: Math.max(storedData.timeElapsed || 0, 0),
        };
        return metrics;
    }

    /**
     * Converts a stored Run into an owner object.
     * 
     * @param storedData The stored data to convert
     * @param userData Session data for the user running the routine
     * @returns The owner object
     */
    private storedDataToOwner(
        storedData: Run,
        userData: RunTriggeredBy,
    ): RunProgress["owner"] {
        if (storedData.team) {
            return { id: storedData.team.id, __typename: "Team" as const };
        }
        return { id: userData.id, __typename: "User" as const };
    }

    /**
     * Shapes a RunStepShape into a RunProgressStep.
     * @param step The run step shape to convert
     * @returns The reconstructed run progress step
     */
    private fromRunStepShape(step: Run["steps"][number]): RunProgressStep {
        let objectType = "Project" as "Project" | "Routine";
        if (step.resourceVersion?.resourceSubType.startsWith("Routine")) {
            objectType = "Routine";
        }
        return {
            completedAt: step.completedAt || undefined,
            complexity: step.complexity || 0,
            contextSwitches: step.contextSwitches || 0,
            id: step.id,
            locationId: step.nodeId,
            name: step.name,
            objectId: step.resourceInId, // The ID of the routine version the node is a part of. May be different than the routine version being run if we step into a nested routine
            objectType,
            order: step.order,
            startedAt: step.startedAt,
            status: step.status,
            subroutineId: step.resourceVersion?.id || null,
        };
    }

    /**
     * Shapes the steps in a stored Run into RunProgressSteps,
     * and sorts them by order.
     * 
     * @param storedData The stored run to convert
     * @returns The reconstructed run progress steps
     */
    private fromRunSteps(storedData: Run): RunProgressStep[] {
        return storedData.steps
            .sort((a, b) => a.order - b.order)
            .map(this.fromRunStepShape);
    }

    /**
     * Converts a stored RunRoutine into a RunProgress object.
     * 
     * @param storedData The stored data to convert
     * @param userData Session data for the user running the routine
     * @param logger The logger to use for any errors
     */
    private fromFetchedRun(
        storedData: Run,
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
        const steps = this.fromRunSteps(storedData);

        const progress: RunProgress = {
            __version,
            branches,
            config: { ...config, isPrivate: storedData.isPrivate },
            decisions,
            name,
            owner,
            runId,
            runOnObjectId: storedData.resourceVersion?.id || null,
            schedule,
            steps,
            metrics,
            status,
            subcontexts,
        };
        return progress;
    }

}

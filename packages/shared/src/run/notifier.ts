import { PERCENTS } from "../consts/numbers.js";
import { SEND_PROGRESS_UPDATE_THROTTLE_MS } from "./consts.js";
import { DeferredDecisionData, IOKey, IOMap, IOValue, Id, MapDiff, RunProgress, RunStatusChangeReason, RunTaskInfo, RunType, SubcontextUpdates } from "./types.js";

/**
 * Handles emitting payloads to send run-related events to the client. This includes:
 * - Throttling `sendProgressUpdate` calls to avoid excessive network traffic
 * - Calculating a diff for the subcontexts to reduce the payload size
 * 
 * When running the state machine in the server, these events should be sent through a WebSocket 
 * for real-time updates in the client.
 * 
 * NOTE: We assume that only one instance of this class is used per run.
 */
export abstract class RunNotifier {
    /**
    * Baseline progress info that was last sent to the client.
    * We keep this around so that on each update we can compute a diff of the subcontexts.
    */
    private _lastSendProgressInfo: {
        progress: RunProgress;
        runStatusChangeReason: RunStatusChangeReason | undefined;
    } | null = null;

    /**
     * A pending progress update that has not yet been flushed.
     */
    private _pendingSendProgressInfo: {
        progress: RunProgress;
        runStatusChangeReason: RunStatusChangeReason | undefined;
    } | null = null;

    /** Throttle timer ID for scheduling the next progress update */
    private _throttleTimer: NodeJS.Timeout | null = null;

    /** Throttle interval for sending progress updates to the client */
    private THROTTLE_INTERVAL_MS = SEND_PROGRESS_UPDATE_THROTTLE_MS;

    /** Timestamp when the last progress update was sent */
    private _lastSent = 0;

    /**
     * Throttled send of the run's current progress to the client.
     * Sends immediately if the throttle interval has passed, otherwise stores the update and
     * schedules it to be sent once the interval elapses.x
     *
     * @param progress The run progress to calculate the payload from
     * @param runStatusChangeReason Optional reason for the run status change, if any.
     */
    public sendProgressUpdate(progress: RunProgress, runStatusChangeReason: RunStatusChangeReason | undefined): void {
        const now = Date.now();
        const elapsed = now - this._lastSent;

        // If enough time has passed, send the update immediately.
        if (elapsed >= this.THROTTLE_INTERVAL_MS) {
            this._lastSent = now;
            const payload = this.createPayload(progress, runStatusChangeReason);
            // Update baseline with the new progress
            this._lastSendProgressInfo = { progress, runStatusChangeReason };
            const { runId, type: runType } = progress;
            this.emitProgressUpdate(runId, runType, payload);
        } else {
            // Otherwise, store the update as pending.
            this._pendingSendProgressInfo = { progress, runStatusChangeReason };

            // Schedule a flush if one isn’t already scheduled.
            if (!this._throttleTimer) {
                const remaining = this.THROTTLE_INTERVAL_MS - elapsed;
                this._throttleTimer = setTimeout(() => {
                    this.flushProgressUpdate();
                }, remaining);
            }
        }
    }

    /**
     * Immediately flushes any pending progress update.
     * If a throttle timer is active, it cancels the timer and sends the pending update.
     * 
     * This is useful for when the run is finished or terminated so that we can send the final update.
     *
     * @returns true when any pending update has been sent.
     */
    public finalizeSend(): boolean {
        if (this._throttleTimer) {
            clearTimeout(this._throttleTimer);
            this.flushProgressUpdate();
        }
        return true;
    }

    /**
     * Flushes any pending progress update by calling the abstract `emitProgressUpdate` method.
     */
    private flushProgressUpdate(): void {
        if (this._pendingSendProgressInfo !== null) {
            const { progress, runStatusChangeReason } = this._pendingSendProgressInfo;
            this._lastSent = Date.now();
            const payload = this.createPayload(progress, runStatusChangeReason);
            // Update the baseline so that subsequent diffs are incremental.
            this._lastSendProgressInfo = { progress, runStatusChangeReason };
            const { runId, type: runType } = progress;
            this.emitProgressUpdate(runId, runType, payload);
            this._pendingSendProgressInfo = null;
        }
        this._throttleTimer = null;
    }

    /**
     * Creates the payload from the provided run progress.
     *
     * @param progress The current run progress
     * @param runStatusChangeReason Optional reason for the run status change, if any.
     * @returns A RunTaskInfo payload that includes subcontextUpdates.
     */
    private createPayload(progress: RunProgress, runStatusChangeReason: RunStatusChangeReason | undefined): RunTaskInfo {
        // Calculate active branches
        const activeBranches: RunTaskInfo["activeBranches"] = progress.branches.map(b => ({
            locationStack: b.locationStack,
            nodeStartTimeMs: b.nodeStartTimeMs,
            processId: b.processId,
            status: b.status,
            subroutineInstanceId: b.subroutineInstanceId,
        }));
        // Calculate percent complete
        const { complexityCompleted, complexityTotal } = progress.metrics;
        const percentComplete = complexityCompleted && complexityTotal ? Math.round(complexityCompleted / complexityTotal * PERCENTS) : 0;
        // Calculate subcontext updates
        const subcontextUpdates = this.calculateSubcontextUpdates(progress);

        const payload: RunTaskInfo = {
            activeBranches,
            percentComplete,
            runId: progress.runId,
            runStatus: progress.status,
            runStatusChangeReason,
            subcontextUpdates,
        };
        return payload;
    }

    /**
     * Calculates the subcontextUpdates diff by comparing the new progress’s subcontexts with the baseline.
     * For each subcontext that’s new or has changes in its allInputsMap or allOutputsMap,
     * we include its entire new values.
     *
     * @param newProgress The new run progress
     * @returns The subcontextUpdates to include in the payload.
     */
    private calculateSubcontextUpdates(newProgress: RunProgress): SubcontextUpdates {
        const updates: SubcontextUpdates = {};
        // Use the baseline subcontexts from the last sent progress; if none, assume an empty baseline.
        const baselineSubcontexts = this._lastSendProgressInfo ? this._lastSendProgressInfo.progress.subcontexts : {};
        const newSubcontexts = newProgress.subcontexts;

        for (const key in newSubcontexts) {
            const newSub = newSubcontexts[key];
            if (!newSub) {
                continue;
            }
            const oldSub = baselineSubcontexts[key];
            if (!oldSub) {
                // New subcontext entirely.
                updates[key] = {
                    allInputsMap: newSub.allInputsMap,
                    allOutputsMap: newSub.allOutputsMap,
                };
            } else {
                // Compute diffs for the IO maps.
                const inputsDiff = this.diffIOMap(newSub.allInputsMap, oldSub.allInputsMap);
                const outputsDiff = this.diffIOMap(newSub.allOutputsMap, oldSub.allOutputsMap);
                // If any changes occurred, include the new values.
                // If this ends up being too much data, we can change this later to only include the keys that changed.
                if (
                    Object.keys(inputsDiff.set).length > 0 ||
                    inputsDiff.removed.length > 0 ||
                    Object.keys(outputsDiff.set).length > 0 ||
                    outputsDiff.removed.length > 0
                ) {
                    updates[key] = {
                        allInputsMap: newSub.allInputsMap,
                        allOutputsMap: newSub.allOutputsMap,
                    };
                }
            }
        }

        return updates;
    }

    /**
     * Compares two IOMap objects and produces a diff.
     *
     * @param newMap The new IOMap
     * @param oldMap The old IOMap
     * @returns A MapDiff showing keys that are new/updated and keys that were removed.
     */
    private diffIOMap(newMap: IOMap, oldMap: IOMap): MapDiff {
        const set: Record<IOKey, IOValue> = {};
        const removed: IOKey[] = [];

        // Check for new or updated keys.
        for (const key in newMap) {
            if (!(key in oldMap) || oldMap[key] !== newMap[key]) {
                set[key] = newMap[key];
            }
        }

        // Check for keys that have been removed.
        for (const key in oldMap) {
            if (!(key in newMap)) {
                removed.push(key);
            }
        }

        return { set, removed };
    }

    /**
    * Sends the progress update to the client.
    *
    * @param runId The ID of the run
    * @param runType The type of run
    * @param payload The payload to send to the client
    */
    protected abstract emitProgressUpdate(runId: Id, runType: RunType, payload: RunTaskInfo): void;

    /**
     * Sends a request to the client to make a decision.
     * 
     * @param runId The ID of the run
     * @param runType The type of run
     * @param decision The decision to make
     */
    public abstract sendDecisionRequest(runId: Id, runType: RunType, decision: DeferredDecisionData): void;
}

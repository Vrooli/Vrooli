import { DUMMY_ID, Run, RunCreateInput, RunStatus, RunTaskInfo, RunUpdateInput, endpointsRun, validatePK } from "@local/shared";
import { useCallback, useEffect, useRef } from "react";
import { fetchLazyWrapper } from "../api/fetchWrapper.js";
import { SocketService } from "../api/socket.js";
import { PubSub } from "../utils/pubsub.js";
import { useLazyFetch } from "./useFetch.js";

type CreateRunProps = Partial<RunCreateInput> & {
    objectId: string;
    objectName: string | null | undefined;
    onSuccess: (data: Run) => unknown;
}

type UpdateProps = RunUpdateInput & {
    onSuccess: (data: Run) => unknown;
}

export function useUpsertRun() {
    const [create, { loading: isCreatingRun }] = useLazyFetch<RunCreateInput, Run>(endpointsRun.createOne);
    const [update, { loading: isUpdatingRun }] = useLazyFetch<RunUpdateInput, Run>(endpointsRun.updateOne);

    const createRun = useCallback(function createRunCallback({
        objectId,
        objectName,
        onSuccess,
        ...rest
    }: CreateRunProps) {
        if (!objectId || !validatePK(objectId)) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }

        fetchLazyWrapper<RunCreateInput, Run>({
            fetch: create,
            inputs: {
                id: DUMMY_ID,
                isPrivate: true,
                name: objectName || "Unnamed Routine",
                resourceVersionConnect: objectId,
                status: RunStatus.InProgress,
                ...rest,
            },
            successCondition: (data) => data !== null,
            onSuccess,
            errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
        });
    }, [create]);

    const updateRun = useCallback(function updateRunCallback({
        onSuccess,
        ...rest
    }: UpdateProps) {
        fetchLazyWrapper<RunUpdateInput, Run>({
            fetch: update,
            inputs: rest,
            successCondition: (data) => data !== null,
            onSuccess,
        });
    }, [update]);

    return {
        createRun,
        isCreatingRun,
        isUpdatingRun,
        updateRun,
    };
}

type UseSocketRunProps = {
    applyRunUpdate: (callback: (existingRun: Run | null) => Run | null) => void;
    runId: string | undefined;
}

export function processRunTaskUpdate(
    applyRunUpdate: UseSocketRunProps["applyRunUpdate"],
    payload: RunTaskInfo,
) {
    // Display errors
    if (payload.runStatus === RunStatus.Failed) {
        // Can use payload.runStatusChangeReason if you want to display a specific error message (just make sure there's a translation key for it)
        PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
    }
    // Update run with new information
    applyRunUpdate((existingRun) => {
        if (!existingRun || existingRun.id !== payload.runId) return existingRun;
        // if (payload.activeNodes) {
        //     //...
        // }
        // if (payload.inputsCreated) {
        //     //...
        // }
        // if (payload.outputsCreated) {
        //     //...
        // }
        // if (payload.percentComplete) {
        //     //...
        // }
        return existingRun;
    });
}

export function useSocketRun({
    applyRunUpdate,
    runId,
}: UseSocketRunProps) {

    // Handle connection/disconnection
    const prevRunId = useRef<string | undefined>(undefined);
    useEffect(function connectToRunEffect() {
        // Make sure we have sufficient data to connect to the run room
        if (!runId || runId === DUMMY_ID) return;
        // Prevent reconnection if the run ID hasn't changed
        if (prevRunId.current === runId) return;
        prevRunId.current = runId;

        SocketService.get().emitEvent("joinRunRoom", { runId }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "RunRoomJoinFailed", severity: "Error" });
            }
        });

        return () => {
            SocketService.get().emitEvent("leaveRunRoom", { runId }, (response) => {
                if (response.error) {
                    console.error("Failed to leave run room", response.error);
                }
            });
        };
    }, [runId]);

    // Handle incoming data
    useEffect(() => SocketService.get().onEvent("runTask", (payload) => processRunTaskUpdate(applyRunUpdate, payload)), [applyRunUpdate]);
}

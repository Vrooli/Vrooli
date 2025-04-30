import { DUMMY_ID, RunProject, RunProjectCreateInput, RunProjectUpdateInput, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput, RunStatus, RunTaskInfo, endpointsRunProject, endpointsRunRoutine, uuid, uuidValidate } from "@local/shared";
import { useCallback, useEffect, useRef } from "react";
import { fetchLazyWrapper } from "../api/fetchWrapper.js";
import { SocketService } from "../api/socket.js";
import { PubSub } from "../utils/pubsub.js";
import { useLazyFetch } from "./useLazyFetch.js";

type CreateRunRoutineProps = Partial<RunRoutineCreateInput> & {
    objectId: string;
    objectName: string | null | undefined;
    onSuccess: (data: RunRoutine) => unknown;
}

type UpdateRunRoutineProps = RunRoutineUpdateInput & {
    onSuccess: (data: RunRoutine) => unknown;
}

export function useUpsertRunRoutine() {
    const [createRunRoutine, { loading: isCreatingRunRoutine }] = useLazyFetch<RunRoutineCreateInput, RunRoutine>(endpointsRunRoutine.createOne);
    const [updateRunRoutine, { loading: isUpdatingRunRoutine }] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointsRunRoutine.updateOne);

    const createRun = useCallback(function createRunCallback({
        objectId,
        objectName,
        onSuccess,
        ...rest
    }: CreateRunRoutineProps) {
        if (!objectId || !uuidValidate(objectId)) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }

        fetchLazyWrapper<RunRoutineCreateInput, RunRoutine>({
            fetch: createRunRoutine,
            inputs: {
                id: uuid(),
                isPrivate: true,
                name: objectName || "Unnamed Routine",
                routineVersionConnect: objectId,
                status: RunStatus.InProgress,
                ...rest,
            },
            successCondition: (data) => data !== null,
            onSuccess,
            errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
        });
    }, [createRunRoutine]);

    const updateRun = useCallback(function updateRunCallback({
        onSuccess,
        ...rest
    }: UpdateRunRoutineProps) {
        fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
            fetch: updateRunRoutine,
            inputs: rest,
            successCondition: (data) => data !== null,
            onSuccess,
        });
    }, [updateRunRoutine]);

    return {
        createRun,
        isCreatingRunRoutine,
        isUpdatingRunRoutine,
        updateRun,
    };
}

type CreateRunProjectProps = Partial<RunProjectCreateInput> & {
    objectId: string;
    objectName: string | null | undefined;
    onSuccess: (data: RunProject) => unknown;
}

type UpdateRunProjectProps = RunProjectUpdateInput & {
    onSuccess: (data: RunProject) => unknown;
}

export function useUpsertRunProject() {
    const [createRunProject, { loading: isCreatingRunProject }] = useLazyFetch<RunProjectCreateInput, RunProject>(endpointsRunProject.createOne);
    const [updateRunProject, { loading: isUpdatingRunProject }] = useLazyFetch<RunProjectUpdateInput, RunProject>(endpointsRunProject.updateOne);

    const createRun = useCallback(function createRunCallback({
        objectId,
        objectName,
        onSuccess,
        ...rest
    }: CreateRunProjectProps) {
        if (!objectId || !uuidValidate(objectId)) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }

        fetchLazyWrapper<RunProjectCreateInput, RunProject>({
            fetch: createRunProject,
            inputs: {
                id: uuid(),
                isPrivate: true,
                name: objectName || "Unnamed Project",
                projectVersionConnect: objectId,
                status: RunStatus.InProgress,
                ...rest,
            },
            successCondition: (data) => data !== null,
            onSuccess,
            errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
        });
    }, [createRunProject]);

    const updateRun = useCallback(function updateRunCallback({
        onSuccess,
        ...rest
    }: UpdateRunProjectProps) {
        fetchLazyWrapper<RunProjectUpdateInput, RunProject>({
            fetch: updateRunProject,
            inputs: rest,
            successCondition: (data) => data !== null,
            onSuccess,
        });
    }, [updateRunProject]);

    return {
        createRun,
        isCreatingRunProject,
        isUpdatingRunProject,
        updateRun,
    };
}

type UseSocketRunProps = {
    applyRunUpdate: (callback: (existingRun: RunProject | RunRoutine | null) => RunProject | RunRoutine | null) => void;
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

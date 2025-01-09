import { DUMMY_ID, RunProject, RunProjectCreateInput, RunProjectUpdateInput, RunRoutine, RunRoutineCreateInput, RunRoutineUpdateInput, RunStatus, RunTaskInfo, RunnableProjectVersion, RunnableRoutineVersion, StartRunTaskInput, Success, TaskStatus, endpointsRunProject, endpointsRunRoutine, endpointsTask, uuid, uuidValidate } from "@local/shared";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { SocketService } from "api/socket";
import { useCallback, useEffect, useRef, useState } from "react";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

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
    getFormValues: () => (object | undefined);
    handleRunUpdate: (run: RunProject | RunRoutine) => unknown;
    run: RunProject | RunRoutine | null | undefined;
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion | null | undefined;
}

export function processRunTaskUpdate(
    handleTaskInfoUpdate: (taskInfo: RunTaskInfo) => unknown,
    handleRunUpdate: (run: RunProject | RunRoutine) => unknown,
    payload: RunTaskInfo,
) {
    if (payload.status === TaskStatus.Failed) {
        PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
    }
    // Update the task info
    handleTaskInfoUpdate(payload);
    // Update the run if needed
    if (payload.run) {
        handleRunUpdate(payload.run);
    }
}

export function useSocketRun({
    getFormValues,
    handleRunUpdate,
    run,
    runnableObject,
}: UseSocketRunProps) {

    // Handle connection/disconnection
    const prevRunId = useRef<string | undefined>(undefined);
    useEffect(function connectToRunEffect() {
        if (!run?.id || run.id === DUMMY_ID) return;
        if (prevRunId.current === run.id) return;
        prevRunId.current = run.id;

        SocketService.get().emitEvent("joinRunRoom", { runId: run.id, runType: run.__typename }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "RunRoomJoinFailed", severity: "Error" });
            }
        });

        return () => {
            SocketService.get().emitEvent("leaveRunRoom", { runId: run.id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave run room", response.error);
                }
            });
        };
    }, [run?.id, run?.__typename]);

    // Store refs for parameters to reduce the number of dependencies of socket event handlers. 
    // This reduces the number of times the socket events are connected/disconnected.
    const runRef = useRef(run);
    runRef.current = run;
    const runnableObjectRef = useRef(runnableObject);
    runnableObjectRef.current = runnableObject;

    const [subroutineTaskInfo, setSubroutineTaskInfo] = useState<RunTaskInfo | null>(null);
    const [startTask, { loading: isGeneratingOutputs }] = useLazyFetch<StartRunTaskInput, Success>(endpointsTask.startRunTask);
    // const [cancelTask] = useLazyFetch<CancelTaskInput, Success>(endpointPostCancelTask);

    const handleRunSubroutine = useCallback(function handleGenerateOutputsCallback() {
        const formValues = getFormValues();
        if (
            !runnableObjectRef.current
            || runnableObjectRef.current.__typename !== "RoutineVersion"
            || !runRef.current
            || runRef.current.__typename !== "RunRoutine"
        ) return;
        fetchLazyWrapper<StartRunTaskInput, Success>({
            fetch: startTask,
            inputs: {
                formValues,
                routineVersionId: runnableObjectRef.current.id,
                runId: runRef.current.id,
            },
            spinnerDelay: null, // Disable spinner since this is a background task
            successCondition: (data) => data && data.success === true,
            errorMessage: () => ({ messageKey: "ActionFailed" }),
            // Socket event should update task data on success, so we don't need to do anything here
        });
    }, [getFormValues, startTask]);

    // Handle incoming data
    useEffect(() => SocketService.get().onEvent("runTask", (payload) => processRunTaskUpdate(setSubroutineTaskInfo, handleRunUpdate, payload)), [handleRunUpdate]);

    return {
        handleRunSubroutine,
        isGeneratingOutputs,
        subroutineTaskInfo,
    };
}

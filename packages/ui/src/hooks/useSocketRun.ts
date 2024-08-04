import { DUMMY_ID, RunProject, RunRoutine, RunTaskInfo, RunnableProjectVersion, RunnableRoutineVersion, StartRunTaskInput, Success, TaskStatus, endpointPostStartRunTask } from "@local/shared";
import { emitSocketEvent, fetchLazyWrapper, onSocketEvent } from "api";
import { FormikProps } from "formik";
import { RefObject, useCallback, useEffect, useRef, useState } from "react";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseSocketRunProps = {
    formikRef: RefObject<FormikProps<object>>;
    handleRunRoutineUpdate: (run: RunRoutine) => unknown;
    run: RunProject | RunRoutine | undefined;
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion;
}

export function processRunTaskUpdate(
    handleTaskInfoUpdate: (taskInfo: RunTaskInfo) => unknown,
    handleRunRoutineUpdate: (run: RunRoutine) => unknown,
    payload: RunTaskInfo,
    run: RunRoutine | RunProject | undefined,
) {
    console.log("qqqq in processRunTaskUpdate", payload);
    if (payload.status === TaskStatus.Failed) {
        PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
    }
    // Update the task info
    handleTaskInfoUpdate(payload);
    // Update the run if needed
    if (!run || run.__typename !== "RunRoutine") return;
    let haveOutputsChanged = false;
    const newOutputs = [...(run.outputs || [])];
    if (Array.isArray(payload.outputsCreate) && payload.outputsCreate.length > 0) {
        newOutputs.push(...payload.outputsCreate);
        haveOutputsChanged = true;
    }
    if (Array.isArray(payload.outputsUpdate) && payload.outputsUpdate.length > 0) {
        payload.outputsUpdate.forEach(output => {
            const index = newOutputs.findIndex(o => o.id === output.id);
            if (index >= 0) {
                newOutputs[index] = output;
                haveOutputsChanged = true;
            }
        });
    }
    if (Array.isArray(payload.outputsDelete) && payload.outputsDelete.length > 0) {
        payload.outputsDelete.forEach(id => {
            const index = newOutputs.findIndex(o => o.id === id);
            if (index >= 0) {
                newOutputs.splice(index, 1);
                haveOutputsChanged = true;
            }
        });
    }
    if (haveOutputsChanged) {
        handleRunRoutineUpdate({ ...run, outputs: newOutputs });
    }
}

export function useSocketRun({
    formikRef,
    handleRunRoutineUpdate,
    run,
    runnableObject,
}: UseSocketRunProps) {

    // Handle connection/disconnection
    useEffect(function connectToRunEffect() {
        if (!run?.id || run.id === DUMMY_ID) return;

        emitSocketEvent("joinRunRoom", { runId: run.id, runType: run.__typename }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "RunRoomJoinFailed", severity: "Error" });
            }
        });

        return () => {
            emitSocketEvent("leaveRunRoom", { runId: run.id }, (response) => {
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
    const [startTask, { loading: isGeneratingOutputs }] = useLazyFetch<StartRunTaskInput, Success>(endpointPostStartRunTask);
    // const [cancelTask] = useLazyFetch<CancelTaskInput, Success>(endpointPostCancelTask);

    const handleRunSubroutine = useCallback(function handleGenerateOutputsCallback() {
        if (runnableObjectRef.current.__typename !== "RoutineVersion" || !runRef.current || runRef.current.__typename !== "RunRoutine") return;
        fetchLazyWrapper<StartRunTaskInput, Success>({
            fetch: startTask,
            inputs: {
                formValues: formikRef.current?.values,
                routineVersionId: runnableObjectRef.current.id,
                runId: runRef.current.id,
            },
            spinnerDelay: null, // Disable spinner since this is a background task
            successCondition: (data) => data && data.success === true,
            errorMessage: () => ({ messageKey: "ActionFailed" }),
            // Socket event should update task data on success, so we don't need to do anything here
        });
    }, [formikRef, startTask]);

    // Handle incoming data
    useEffect(() => onSocketEvent("runTask", (payload) => processRunTaskUpdate(setSubroutineTaskInfo, handleRunRoutineUpdate, payload, runRef.current)), []);

    return {
        handleRunSubroutine,
        isGeneratingOutputs,
        subroutineTaskInfo,
    };
}

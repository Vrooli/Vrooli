import { DUMMY_ID, RunProject, RunRoutine, RunTaskInfo, RunnableProjectVersion, RunnableRoutineVersion, StartRunTaskInput, Success, TaskStatus, endpointPostStartRunTask } from "@local/shared";
import { emitSocketEvent, fetchLazyWrapper, onSocketEvent } from "api";
import { FormikProps } from "formik";
import { RefObject, useCallback, useEffect, useRef, useState } from "react";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UseSocketRunProps = {
    formikRef: RefObject<FormikProps<object>>;
    handleRunUpdate: (run: RunProject | RunRoutine) => unknown;
    run: RunProject | RunRoutine | undefined;
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion;
}

export function processRunTaskUpdate(
    handleTaskInfoUpdate: (taskInfo: RunTaskInfo) => unknown,
    handleRunUpdate: (run: RunProject | RunRoutine) => unknown,
    payload: RunTaskInfo,
) {
    console.log("qqqq in processRunTaskUpdate", payload);
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
    formikRef,
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
    useEffect(() => onSocketEvent("runTask", (payload) => processRunTaskUpdate(setSubroutineTaskInfo, handleRunUpdate, payload)), [handleRunUpdate]);

    return {
        handleRunSubroutine,
        isGeneratingOutputs,
        subroutineTaskInfo,
    };
}

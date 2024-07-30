import { DUMMY_ID, RunProject, RunRoutine, RunSocketEventPayloads } from "@local/shared";
import { emitSocketEvent, onSocketEvent } from "api";
import { useEffect, useRef } from "react";
import { PubSub } from "utils/pubsub";

type UseSocketRunProps = {
    run: RunProject | RunRoutine | undefined;
}

export function processRunStepOutput(
    result: RunSocketEventPayloads["runResult"],
) {
    //TODO
}

export function useSocketRun({
    run,
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

    // Handle incoming data
    useEffect(() => onSocketEvent("runResult", (payload) => processRunStepOutput(payload)), []);
}

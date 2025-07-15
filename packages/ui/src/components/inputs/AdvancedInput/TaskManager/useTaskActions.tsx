import { useCallback } from "react";
import { AITaskDisplayState, type AITaskDisplay } from "../../../../types.js";

export function useTaskActions(tasks: AITaskDisplay[], onTasksChange?: (updatedTools: AITaskDisplay[]) => unknown) {
    const handleToggleTask = useCallback(
        (index: number) => {
            if (!onTasksChange) return;
            const updated = [...tasks];
            const currentState = updated[index].state;
            let newState: AITaskDisplayState;
            if (currentState === AITaskDisplayState.Disabled) {
                newState = AITaskDisplayState.Enabled;
            } else if (currentState === AITaskDisplayState.Enabled) {
                newState = AITaskDisplayState.Disabled;
            } else {
                // Exclusive -> Enabled
                newState = AITaskDisplayState.Enabled;
            }
            updated[index] = { ...updated[index], state: newState };
            onTasksChange(updated);
        },
        [tasks, onTasksChange],
    );

    const handleToggleTaskExclusive = useCallback(
        (index: number) => {
            if (!onTasksChange) return;
            let updated = [...tasks];
            // If the task is already exclusive, set it to enabled
            const currentState = tasks[index].state;
            if (currentState === AITaskDisplayState.Exclusive) {
                updated = tasks.map((task, i) => {
                    if (i === index) {
                        return { ...task, state: AITaskDisplayState.Enabled };
                    } else {
                        return task;
                    }
                });
            }
            // Otherwise, set the task to exclusive and any existing exclusives to enabled
            else {
                updated = tasks.map((task, i) => {
                    if (i === index) {
                        return { ...task, state: AITaskDisplayState.Exclusive };
                    } else if (task.state === AITaskDisplayState.Exclusive) {
                        return { ...task, state: AITaskDisplayState.Enabled };
                    } else {
                        return task;
                    }
                });
            }
            onTasksChange(updated);
        },
        [tasks, onTasksChange],
    );

    const handleRemoveTask = useCallback(
        (index: number) => {
            if (!onTasksChange) return;
            const updated = [...tasks];
            updated.splice(index, 1);
            onTasksChange(updated);
        },
        [tasks, onTasksChange],
    );

    return { handleToggleTask, handleToggleTaskExclusive, handleRemoveTask };
}

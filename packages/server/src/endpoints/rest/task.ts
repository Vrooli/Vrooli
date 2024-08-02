import { task_autoFill, task_cancelTask, task_checkTaskStatuses, task_startLlmTask, task_startRunTask } from "../generated";
import { TaskEndpoints } from "../logic/task";
import { setupRoutes } from "./base";

export const TaskRest = setupRoutes({
    "/autoFill": {
        get: [TaskEndpoints.Mutation.autoFill, task_autoFill],
    },
    "/startLlmTask": {
        post: [TaskEndpoints.Mutation.startLlmTask, task_startLlmTask],
    },
    "/startRunTask": {
        post: [TaskEndpoints.Mutation.startRunTask, task_startRunTask],
    },
    "/cancelTask": {
        post: [TaskEndpoints.Mutation.cancelTask, task_cancelTask],
    },
    "/checkTaskStatuses": {
        get: [TaskEndpoints.Query.checkTaskStatuses, task_checkTaskStatuses],
    },
});

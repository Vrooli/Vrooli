import { task_autoFill, task_cancelTask, task_checkTaskStatuses, task_startTask } from "../generated";
import { TaskEndpoints } from "../logic/task";
import { setupRoutes } from "./base";

export const TaskRest = setupRoutes({
    "/autoFill": {
        get: [TaskEndpoints.Mutation.autoFill, task_autoFill],
    },
    "/startTask": {
        post: [TaskEndpoints.Mutation.startTask, task_startTask],
    },
    "/cancelTask": {
        post: [TaskEndpoints.Mutation.cancelTask, task_cancelTask],
    },
    "/checkTaskStatuses": {
        get: [TaskEndpoints.Query.checkTaskStatuses, task_checkTaskStatuses],
    },
});

import { type CancelTaskInput, type CheckTaskStatusesInput, type CheckTaskStatusesResult, nanoid, RunTriggeredFrom, type StartLlmTaskInput, type StartRunTaskInput, type Success, TaskType } from "@local/shared";
import { RequestService } from "../../auth/request.js";
import { changeRunTaskStatus, getRunTaskStatuses, processRun } from "../../tasks/run/queue.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue.js";
import { type ApiEndpoint } from "../../types.js";

export type EndpointsTask = {
    checkStatuses: ApiEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    startLlmTask: ApiEndpoint<StartLlmTaskInput, Success>;
    startRunTask: ApiEndpoint<StartRunTaskInput, Success>;
    cancelTask: ApiEndpoint<CancelTaskInput, Success>;
}

export const task: EndpointsTask = {
    checkStatuses: async ({ input }, { req }) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const result: CheckTaskStatusesResult = {
            __typename: "CheckTaskStatusesResult",
            statuses: [],
        };
        switch (input.taskType) {
            case TaskType.Llm:
                result.statuses = await getLlmTaskStatuses(input.taskIds);
                break;
            case TaskType.Run:
                result.statuses = await getRunTaskStatuses(input.taskIds);
                break;
            case TaskType.Sandbox:
                result.statuses = await getSandboxTaskStatuses(input.taskIds);
                break;
        }
        return result;
    },
    startLlmTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });

        //TODO
        // return QueueService.get().llm.addTask({
        //     type: QueueTaskType.LLM_COMPLETION,
        //     id: input.parentId.messageId, // Use messageId so we override any existing task for the same message
        //     chatId: input.chatId,
        //     messageId: input.parentId
        // });
        return {} as any;
    },
    startRunTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });

        const taskId = `task-${nanoid()}`;
        return processRun({
            ...input,
            runFrom: RunTriggeredFrom.RunView, // Can customize this later to change queue priority
            startedById: userData.id,
            id: taskId,
            userData,
        });
    },
    cancelTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });

        switch (input.taskType) {
            case TaskType.Llm:
                return changeLlmTaskStatus(input.taskId, "Suggested", userData.id);
            case TaskType.Run:
                return changeRunTaskStatus(input.taskId, "Suggested", userData.id);
            case TaskType.Sandbox:
                return changeSandboxTaskStatus(input.taskId, "Suggested", userData.id);
        }
    },
};

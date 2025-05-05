import { CancelTaskInput, CheckTaskStatusesInput, CheckTaskStatusesResult, nanoid, RunTriggeredFrom, StartLlmTaskInput, StartRunTaskInput, Success, TaskType } from "@local/shared";
import { RequestService } from "../../auth/request.js";
import { requestBotResponse } from "../../tasks/llm/queue.js";
import { changeLlmTaskStatus, getLlmTaskStatuses } from "../../tasks/llmTask/queue.js";
import { changeRunTaskStatus, getRunTaskStatuses, processRun } from "../../tasks/run/queue.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue.js";
import { ApiEndpoint } from "../../types.js";

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

        return requestBotResponse({
            ...input,
            mode: "json",
            parentMessage: null,
            userData,
        });
    },
    startRunTask: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });

        const taskId = `task-${nanoid()}`;
        return processRun({
            ...input,
            runFrom: RunTriggeredFrom.RunView, // Can customize this later to change queue priority
            startedById: userData.id,
            taskId,
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

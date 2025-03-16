import { CancelTaskInput, CheckTaskStatusesInput, CheckTaskStatusesResult, RunTriggeredFrom, StartLlmTaskInput, StartRunTaskInput, Success, TaskType, uuid } from "@local/shared";
import { RequestService } from "../../auth/request.js";
import { requestBotResponse } from "../../tasks/llm/queue.js";
import { changeLlmTaskStatus, getLlmTaskStatuses } from "../../tasks/llmTask/queue.js";
import { changeRunTaskStatus, getRunTaskStatuses, processRunProject, processRunRoutine } from "../../tasks/run/queue.js";
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

        const taskId = `task-${uuid()}`;
        const { projectVersionId, routineVersionId } = input;
        if (projectVersionId) {
            return processRunProject({
                ...input,
                projectVersionId,
                runFrom: RunTriggeredFrom.RunView, // Can customize this later to change queue priority
                startedById: userData.id,
                taskId,
                type: "RunProject",
                userData,
            });
        } else if (routineVersionId) {
            return processRunRoutine({
                ...input,
                routineVersionId,
                runFrom: RunTriggeredFrom.RunView, // Can customize this later to change queue priority
                startedById: userData.id,
                taskId,
                type: "RunRoutine",
                userData,
            });
        } else {
            return { __typename: "Success", success: false } as const;
        }
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

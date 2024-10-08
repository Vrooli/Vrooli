import { CancelTaskInput, CheckTaskStatusesInput, CheckTaskStatusesResult, RunFrom, StartLlmTaskInput, StartRunTaskInput, Success, TaskType, uuid } from "@local/shared";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { requestBotResponse } from "../../tasks/llm/queue";
import { changeLlmTaskStatus, getLlmTaskStatuses } from "../../tasks/llmTask";
import { changeRunTaskStatus, getRunTaskStatuses, processRunProject, processRunRoutine } from "../../tasks/run/queue";
import { changeSandboxTaskStatus, getSandboxTaskStatuses } from "../../tasks/sandbox/queue";
import { GQLEndpoint } from "../../types";

export type EndpointsTask = {
    Query: {
        checkTaskStatuses: GQLEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    },
    Mutation: {
        startLlmTask: GQLEndpoint<StartLlmTaskInput, Success>;
        startRunTask: GQLEndpoint<StartRunTaskInput, Success>;
        cancelTask: GQLEndpoint<CancelTaskInput, Success>;
    }
}

export const TaskEndpoints: EndpointsTask = {
    Query: {
        checkTaskStatuses: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });
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
    },
    Mutation: {
        startLlmTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return requestBotResponse({
                ...input,
                mode: "json",
                parentMessage: null,
                userData
            });
        },
        startRunTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            const taskId = `task-${uuid()}`;
            const projectVersionId = input.projectVerisonId;
            const routineVersionId = input.routineVersionId;
            if (projectVersionId) {
                return processRunProject({
                    ...input,
                    projectVersionId,
                    taskId,
                    runFrom: RunFrom.RunView, // Can customize this later to change queue priority
                    runType: "RunProject",
                    startedById: userData.id,
                    userData,
                });
            } else if (routineVersionId) {
                return processRunRoutine({
                    ...input,
                    routineVersionId,
                    runFrom: RunFrom.RunView, // Can customize this later to change queue priority
                    runType: "RunRoutine",
                    taskId,
                    startedById: userData.id,
                    userData,
                });
            } else {
                return { __typename: "Success", success: false } as const;
            }
        },
        cancelTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            switch (input.taskType) {
                case TaskType.Llm:
                    return changeLlmTaskStatus(input.taskId, "Suggested", userData.id);
                case TaskType.Run:
                    return changeRunTaskStatus(input.taskId, "Suggested", userData.id);
                case TaskType.Sandbox:
                    return changeSandboxTaskStatus(input.taskId, "Suggested", userData.id);
            }
        },
    },
};

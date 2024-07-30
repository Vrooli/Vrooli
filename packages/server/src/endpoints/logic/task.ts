import { AutoFillInput, AutoFillResult, CancelTaskInput, CheckTaskStatusesInput, CheckTaskStatusesResult, StartTaskInput, Success } from "@local/shared";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { llmProcessAutoFill } from "../../tasks/llm/process";
import { requestStartTask } from "../../tasks/llm/queue";
import { changeLlmTaskStatus, getLlmTaskStatuses } from "../../tasks/llmTask";
import { GQLEndpoint } from "../../types";

export type EndpointsTask = {
    Query: {
        checkTaskStatuses: GQLEndpoint<CheckTaskStatusesInput, CheckTaskStatusesResult>;
    },
    Mutation: {
        autoFill: GQLEndpoint<AutoFillInput, AutoFillResult>;
        startTask: GQLEndpoint<StartTaskInput, Success>;
        cancelTask: GQLEndpoint<CancelTaskInput, Success>;
    }
}

export const TaskEndpoints: EndpointsTask = {
    Query: {
        checkTaskStatuses: async (_, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });

            const statuses = await getLlmTaskStatuses(input.taskIds);
            return { __typename: "CheckTaskStatusesResult", statuses };
        },
    },
    Mutation: {
        autoFill: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return llmProcessAutoFill({ ...input, userData, __process: "AutoFill" });
        },
        startTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return requestStartTask({ ...input, userData });
        },
        cancelTask: async (_, { input }, { req }) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });

            return changeLlmTaskStatus(input.taskId, "Suggested", userData.id);
        },
    },
};

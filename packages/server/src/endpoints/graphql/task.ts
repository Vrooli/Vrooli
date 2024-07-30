import { LlmTask, LlmTaskStatus } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsTask, TaskEndpoints } from "../logic/task";

export const typeDef = gql`
    enum LlmTask {
        ApiAdd
        ApiDelete
        ApiFind
        ApiUpdate
        BotAdd
        BotDelete
        BotFind
        BotUpdate
        DataConverterAdd
        DataConverterDelete
        DataConverterFind
        DataConverterUpdate
        MembersAdd
        MembersDelete
        MembersFind
        MembersUpdate
        NoteAdd
        NoteDelete
        NoteFind
        NoteUpdate
        ProjectAdd
        ProjectDelete
        ProjectFind
        ProjectUpdate
        ReminderAdd
        ReminderDelete
        ReminderFind
        ReminderUpdate
        RoleAdd
        RoleDelete
        RoleFind
        RoleUpdate
        RoutineAdd
        RoutineDelete
        RoutineFind
        RoutineUpdate
        RunProjectStart
        RunRoutineStart
        ScheduleAdd
        ScheduleDelete
        ScheduleFind
        ScheduleUpdate
        SmartContractAdd
        SmartContractDelete
        SmartContractFind
        SmartContractUpdate
        StandardAdd
        StandardDelete
        StandardFind
        StandardUpdate
        Start
        TeamAdd
        TeamDelete
        TeamFind
        TeamUpdate
    }

    enum LlmTaskStatus {
        Canceling
        Completed
        Failed
        Running
        Scheduled
        Suggested
    }

    input AutoFillInput {
        task: LlmTask!
        data: JSON!
    }

    type AutoFillResult {
        data: JSON!
    }

    input StartTaskInput {
        botId: String!
        label: String!
        messageId: ID!
        properties: JSON!
        task: LlmTask!
        taskId: ID!
    }

    input CancelTaskInput {
        taskId: ID!
    }

    input CheckTaskStatusesInput {
        taskIds: [ID!]!
    }

    type LlmTaskStatusInfo {
        id: ID!
        status: LlmTaskStatus
    }

    type CheckTaskStatusesResult {
        statuses: [LlmTaskStatusInfo!]!
    }

    extend type Query {
        checkTaskStatuses(input: CheckTaskStatusesInput!): CheckTaskStatusesResult!
    }

    extend type Mutation {
        autoFill(input: AutoFillInput!): AutoFillResult!
        startTask(input: StartTaskInput!): Success!
        cancelTask(input: CancelTaskInput!): Success!
    }
`;

export const resolvers: {
    LlmTask: typeof LlmTask;
    LlmTaskStatus: typeof LlmTaskStatus;
    Query: EndpointsTask["Query"];
    Mutation: EndpointsTask["Mutation"];
} = {
    LlmTask,
    LlmTaskStatus,
    ...TaskEndpoints,
};

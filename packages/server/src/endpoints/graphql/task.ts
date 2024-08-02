import { LlmTask, RunTask, SandboxTask, TaskStatus, TaskType } from "@local/shared";
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

    enum RunTask {
        RunProject
        RunRoutine
    }

    enum SandboxTask {
        CallApi
        RunDataTransform
        RunSmartContract
    }

    enum TaskStatus {
        Canceling
        Completed
        Failed
        Running
        Scheduled
        Suggested
    }

    enum TaskType {
        Llm
        Run
        Sandbox
    }

    input AutoFillInput {
        task: LlmTask!
        data: JSON!
    }

    type AutoFillResult {
        data: JSON!
    }

    input StartLlmTaskInput {
        # The ID of the bot the task will be performed by
        botId: String!
        # Label for the task, to provide in notifications
        label: String!
        # The ID of the message the task was suggested in. Used to grab the relevant chat context
        messageId: ID!
        # Any properties provided with the task
        properties: JSON!
        # The task to start
        task: LlmTask!
        # The ID of the task, so we can update its status in the UI
        taskId: ID!
    }

    input StartRunTaskInput {
        inputs: JSON
        projectVerisonId: ID
        routineVersionId: ID
        runId: ID!
    }

    input CancelTaskInput {
        taskId: ID!
        taskType: TaskType!
    }

    input CheckTaskStatusesInput {
        taskIds: [ID!]!
        taskType: TaskType!
    }

    type TaskStatusInfo {
        id: ID!
        status: TaskStatus
    }

    type CheckTaskStatusesResult {
        statuses: [TaskStatusInfo!]!
    }

    extend type Query {
        checkTaskStatuses(input: CheckTaskStatusesInput!): CheckTaskStatusesResult!
    }

    extend type Mutation {
        autoFill(input: AutoFillInput!): AutoFillResult!
        startLlmTask(input: StartLlmTaskInput!): Success!
        startRunTask(input: StartRunTaskInput!): Success!
        cancelTask(input: CancelTaskInput!): Success!
    }
`;

export const resolvers: {
    LlmTask: typeof LlmTask;
    RunTask: typeof RunTask;
    SandboxTask: typeof SandboxTask;
    TaskStatus: typeof TaskStatus;
    TaskType: typeof TaskType;
    Query: EndpointsTask["Query"];
    Mutation: EndpointsTask["Mutation"];
} = {
    LlmTask,
    RunTask,
    SandboxTask,
    TaskStatus,
    TaskType,
    ...TaskEndpoints,
};

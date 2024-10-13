import { LlmTask, RunTask, SandboxTask, TaskStatus, TaskType } from "@local/shared";
import { EndpointsTask, TaskEndpoints } from "../logic/task";

export const typeDef = `#graphql
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
        QuestionAdd
        QuestionDelete
        QuestionFind
        QuestionUpdate
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

    input TaskContextInfoTemplateVariablesInput {
        data: String
        task: String
    }

    input TaskContextInfoInput {
        id: ID!
        # Data can be of any type, so we use JSON
        data: JSON!
        label: String!
        template: String
        templateVariables: TaskContextInfoTemplateVariablesInput
    }

    # See RequestBotMessagePayload for property descriptions
    input StartLlmTaskInput {
        chatId: ID!
        parentId: ID
        respondingBotId: ID!
        shouldNotRunTasks: Boolean!
        task: LlmTask!
        taskContexts: [TaskContextInfoInput!]!
    }

    input StartRunTaskInput {
        formValues: JSON
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

import { Count, FindByIdInput, RunRoutine, RunRoutineCancelInput, RunRoutineCompleteInput, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineSortBy, RunRoutineUpdateInput, VisibilityType } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { assertRequestFrom } from "../auth/request";
import { rateLimit } from "../middleware";
import { RunRoutineModel } from "../models";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from "../types";

export const typeDef = gql`
    enum RunRoutineSortBy {
        ContextSwitchesAsc
        ContextSwitchesDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateStartedAsc
        DateStartedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StepsAsc
        StepsDesc
    }

    input RunRoutineCreateInput {
        id: ID!
        isPrivate: Boolean
        completedComplexity: Int
        contextSwitches: Int
        name: String!
        status: RunStatus!
        stepsCreate: [RunRoutineStepCreateInput!]
        inputsCreate: [RunRoutineInputCreateInput!]
        scheduleCreate: ScheduleCreateInput
        routineVersionConnect: ID!
        runProjectConnect: ID
        organizationConnect: ID
    }
    input RunRoutineUpdateInput {
        id: ID!
        isPrivate: Boolean
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        status: RunStatus
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunRoutineStepCreateInput!]
        stepsUpdate: [RunRoutineStepUpdateInput!]
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
    }
    type RunRoutine {
        id: ID!
        isPrivate: Boolean!
        completedComplexity: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        name: String!
        status: RunStatus!
        wasRunAutomatically: Boolean!
        routineVersion: RoutineVersion
        runProject: RunProject
        schedule: Schedule
        steps: [RunRoutineStep!]!
        stepsCount: Int!
        inputs: [RunRoutineInput!]!
        inputsCount: Int!
        user: User
        organization: Organization
        you: RunRoutineYou!
    }

    type RunRoutineYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
    }

    input RunRoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        scheduleStartTimeFrame: TimeFrame
        scheduleEndTimeFrame: TimeFrame
        status: RunStatus
        statuses: [RunStatus!]
        routineVersionId: ID
        searchString: String
        sortBy: RunRoutineSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunRoutineSearchResult {
        pageInfo: PageInfo!
        edges: [RunRoutineEdge!]!
    }

    type RunRoutineEdge {
        cursor: String!
        node: RunRoutine!
    }

    input RunRoutineCompleteInput {
        id: ID! # Run ID if "exists" is true, or routine version ID if "exists" is false
        completedComplexity: Int # Even though the runRoutine was completed, the user may not have completed every subroutine
        exists: Boolean # If true, runRoutine ID is provided, otherwise routine ID so we can create a runRoutine
        name: String # Title of routine, so runRoutine name stays consistent even if routine updates/deletes
        finalStepCreate: RunRoutineStepCreateInput
        finalStepUpdate: RunRoutineStepUpdateInput
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
        routineVersionConnect: ID # Only needed if "exists" is false
        wasSuccessful: Boolean
    }
    input RunRoutineCancelInput {
        id: ID!
    }

    extend type Query {
        runRoutine(input: FindByIdInput!): RunRoutine
        runRoutines(input: RunRoutineSearchInput!): RunRoutineSearchResult!
    }

    extend type Mutation {
        runRoutineCreate(input: RunRoutineCreateInput!): RunRoutine!
        runRoutineUpdate(input: RunRoutineUpdateInput!): RunRoutine!
        runRoutineDeleteAll: Count!
        runRoutineComplete(input: RunRoutineCompleteInput!): RunRoutine!
        runRoutineCancel(input: RunRoutineCancelInput!): RunRoutine!
    }
`;

const objectType = "RunRoutine";
export const resolvers: {
    RunRoutineSortBy: typeof RunRoutineSortBy;
    Query: {
        runRoutine: GQLEndpoint<FindByIdInput, FindOneResult<RunRoutine>>;
        runRoutines: GQLEndpoint<RunRoutineSearchInput, FindManyResult<RunRoutine>>;
    },
    Mutation: {
        runRoutineCreate: GQLEndpoint<RunRoutineCreateInput, CreateOneResult<RunRoutine>>;
        runRoutineUpdate: GQLEndpoint<RunRoutineUpdateInput, UpdateOneResult<RunRoutine>>;
        runRoutineDeleteAll: GQLEndpoint<Record<string, never>, Count>;
        runRoutineComplete: GQLEndpoint<RunRoutineCompleteInput, RecursivePartial<RunRoutine>>;
        runRoutineCancel: GQLEndpoint<RunRoutineCancelInput, RecursivePartial<RunRoutine>>;
    }
} = {
    RunRoutineSortBy,
    Query: {
        runRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        runRoutineCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runRoutineUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runRoutineDeleteAll: async (_p, _d, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 25, req });
            return RunRoutineModel.danger.deleteAll(prisma, { __typename: "User", id: userData.id });
        },
        runRoutineComplete: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunRoutineModel.run.complete(prisma, userData, input, info);
        },
        runRoutineCancel: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunRoutineModel.run.cancel(prisma, userData, input, info);
        },
    },
};

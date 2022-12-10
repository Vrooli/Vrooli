import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from '../types';
import { Count, FindByIdInput, RunRoutine, RunRoutineCancelInput, RunRoutineCompleteInput, RunRoutineCreateInput, RunRoutineSearchInput, RunRoutineSortBy, RunRoutineUpdateInput, RunStatus } from './types';
import { rateLimit } from '../middleware';
import { assertRequestFrom } from '../auth/request';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { RunRoutineModel } from '../models';

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
        RunRoutinesAsc
        RunRoutinesDesc
        StepsAsc
        StepsDesc
    }

    enum RunStatus {
        Scheduled
        InProgress
        Completed
        Failed
        Cancelled
    }

    type RunRoutine {
        id: ID!
        completedComplexity: Int!
        contextSwitches: Int!
        isPrivate: Boolean!
        permissionsRun: RunRoutinePermission!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        name: String!
        status: RunStatus!
        routine: Routine
        steps: [RunRoutineStep!]!
        inputs: [RunRoutineInput!]!
        user: User!
    }

    type RunRoutinePermission {
        canDelete: Boolean!
        canEdit: Boolean!
        canView: Boolean!
    }

    input RunRoutineSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        status: RunStatus
        routineId: ID
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

    input RunRoutineCreateInput {
        id: ID!
        isPrivate: Boolean
        routineVersionId: ID!
        name: String!
        stepsCreate: [RunRoutineStepCreateInput!]
        inputsCreate: [RunRoutineInputCreateInput!]
        # If scheduling info provided, not starting immediately
        # TODO
    }

    input RunRoutineUpdateInput {
        id: ID!
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        isPrivate: Boolean
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunRoutineStepCreateInput!]
        stepsUpdate: [RunRoutineStepUpdateInput!]
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
    }

    input RunRoutineCompleteInput {
        id: ID! # Run ID if "exists" is true, or routine version ID if "exists" is false
        completedComplexity: Int # Even though the runRoutine was completed, the user may not have completed every subroutine
        exists: Boolean # If true, runRoutine ID is provided, otherwise routine ID so we can create a runRoutine
        name: String! # Title of routine, so runRoutine name stays consistent even if routine updates/deletes
        finalStepCreate: RunRoutineStepCreateInput
        finalStepUpdate: RunRoutineStepUpdateInput
        inputsDelete: [ID!]
        inputsCreate: [RunRoutineInputCreateInput!]
        inputsUpdate: [RunRoutineInputUpdateInput!]
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
`

const objectType = 'RunRoutine';
export const resolvers: {
    RunRoutineSortBy: typeof RunRoutineSortBy;
    RunStatus: typeof RunStatus;
    Query: {
        runRoutine: GQLEndpoint<FindByIdInput, FindOneResult<RunRoutine>>;
        runRoutines: GQLEndpoint<RunRoutineSearchInput, FindManyResult<RunRoutine>>;
    },
    Mutation: {
        runRoutineCreate: GQLEndpoint<RunRoutineCreateInput, CreateOneResult<RunRoutine>>;
        runRoutineUpdate: GQLEndpoint<RunRoutineUpdateInput, UpdateOneResult<RunRoutine>>;
        runRoutineDeleteAll: GQLEndpoint<{}, Count>;
        runRoutineComplete: GQLEndpoint<RunRoutineCompleteInput, RecursivePartial<RunRoutine>>;
        runRoutineCancel: GQLEndpoint<RunRoutineCancelInput, RecursivePartial<RunRoutine>>;
    }
} = {
    RunRoutineSortBy,
    RunStatus,
    Query: {
        runRoutine: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runRoutines: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            const userData = assertRequestFrom(req, { isUser: true });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
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
            return RunRoutineModel.danger.deleteAll(prisma, { __typename: 'User', id: userData.id });
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
    }
}
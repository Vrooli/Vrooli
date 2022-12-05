import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from '../types';
import { Count, FindByIdInput, Run, RunCancelInput, RunCompleteInput, RunCreateInput, RunSearchInput, RunSortBy, RunStatus, RunStepStatus, RunUpdateInput } from './types';
import { rateLimit } from '../middleware';
import { assertRequestFrom } from '../auth/request';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { RunModel } from '../models';

export const typeDef = gql`
    enum RunSortBy {
        DateStartedAsc
        DateStartedDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum RunStatus {
        Scheduled
        InProgress
        Completed
        Failed
        Cancelled
    }

    enum RunStepStatus {
        InProgress
        Completed
        Skipped
    }

    type Run {
        id: ID!
        completedComplexity: Int!
        contextSwitches: Int!
        isPrivate: Boolean!
        permissionsRun: RunPermission!
        timeStarted: Date
        timeElapsed: Int
        timeCompleted: Date
        title: String!
        status: RunStatus!
        routine: Routine
        steps: [RunStep!]!
        inputs: [RunInput!]!
        user: User!
    }

    type RunPermission {
        canDelete: Boolean!
        canEdit: Boolean!
        canView: Boolean!
    }

    type RunStep {
        id: ID!
        order: Int!
        contextSwitches: Int!
        timeStarted: Date
        timeElapsed: Int
        timeCompleted: Date
        title: String!
        status: RunStepStatus!
        step: [Int!]!
        run: Run!
        node: Node
        subroutine: Routine
    }

    input RunSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        status: RunStatus
        routineId: ID
        searchString: String
        sortBy: RunSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunSearchResult {
        pageInfo: PageInfo!
        edges: [RunEdge!]!
    }

    type RunEdge {
        cursor: String!
        node: Run!
    }

    input RunCreateInput {
        id: ID!
        isPrivate: Boolean
        routineVersionId: ID!
        title: String!
        stepsCreate: [RunStepCreateInput!]
        inputsCreate: [RunInputCreateInput!]
        # If scheduling info provided, not starting immediately
        # TODO
    }

    input RunUpdateInput {
        id: ID!
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        isPrivate: Boolean
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunStepCreateInput!]
        stepsUpdate: [RunStepUpdateInput!]
        inputsDelete: [ID!]
        inputsCreate: [RunInputCreateInput!]
        inputsUpdate: [RunInputUpdateInput!]
    }

    input RunStepCreateInput {
        id: ID!
        nodeId: ID
        contextSwitches: Int
        subroutineVersionId: ID
        order: Int!
        step: [Int!]!
        timeElapsed: Int
        title: String!
    }

    input RunStepUpdateInput {
        id: ID!
        contextSwitches: Int
        status: RunStepStatus
        timeElapsed: Int
    }

    input RunCompleteInput {
        id: ID! # Run ID if "exists" is true, or routine version ID if "exists" is false
        completedComplexity: Int # Even though the run was completed, the user may not have completed every subroutine
        exists: Boolean # If true, run ID is provided, otherwise routine ID so we can create a run
        title: String! # Title of routine, so run name stays consistent even if routine updates/deletes
        finalStepCreate: RunStepCreateInput
        finalStepUpdate: RunStepUpdateInput
        inputsDelete: [ID!]
        inputsCreate: [RunInputCreateInput!]
        inputsUpdate: [RunInputUpdateInput!]
        wasSuccessful: Boolean
    }

    input RunCancelInput {
        id: ID!
    }

    extend type Query {
        run(input: FindByIdInput!): Run
        runs(input: RunSearchInput!): RunSearchResult!
    }

    extend type Mutation {
        runCreate(input: RunCreateInput!): Run!
        runUpdate(input: RunUpdateInput!): Run!
        runDeleteAll: Count!
        runComplete(input: RunCompleteInput!): Run!
        runCancel(input: RunCancelInput!): Run!
    }
`

const objectType = 'RunRoutine';
export const resolvers: {
    RunSortBy: typeof RunSortBy;
    RunStatus: typeof RunStatus;
    RunStepStatus: typeof RunStepStatus;
    Query: {
        run: GQLEndpoint<FindByIdInput, FindOneResult<Run>>;
        runs: GQLEndpoint<RunSearchInput, FindManyResult<Run>>;
    },
    Mutation: {
        runCreate: GQLEndpoint<RunCreateInput, CreateOneResult<Run>>;
        runUpdate: GQLEndpoint<RunUpdateInput, UpdateOneResult<Run>>;
        runDeleteAll: GQLEndpoint<{}, Count>;
        runComplete: GQLEndpoint<RunCompleteInput, RecursivePartial<Run>>;
        runCancel: GQLEndpoint<RunCancelInput, RecursivePartial<Run>>;
    }
} = {
    RunSortBy,
    RunStatus,
    RunStepStatus,
    Query: {
        run: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runs: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            const userData = assertRequestFrom(req, { isUser: true });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        runCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runDeleteAll: async (_p, _d, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 25, req });
            return RunModel.danger.deleteAll(prisma, { __typename: 'User', id: userData.id });
        },
        runComplete: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunModel.run.complete(prisma, userData, input, info);
        },
        runCancel: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunModel.run.cancel(prisma, userData, input, info);
        },
    }
}
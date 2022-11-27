import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, Run, RunCancelInput, RunCompleteInput, RunCountInput, RunCreateInput, RunSearchInput, RunSearchResult, RunSortBy, RunStatus, RunStepStatus, RunUpdateInput } from './types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { assertRequestFrom } from '../auth/request';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
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

    # Return type for search result
    type RunSearchResult {
        pageInfo: PageInfo!
        edges: [RunEdge!]!
    }

    # Return type for search result edge
    type RunEdge {
        cursor: String!
        node: Run!
    }

    # Input for count
    input RunCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
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
        runsCount(input: RunCountInput!): Int!
    }

    extend type Mutation {
        runCreate(input: RunCreateInput!): Run!
        runUpdate(input: RunUpdateInput!): Run!
        runDeleteAll: Count!
        runDeleteMany(input: DeleteManyInput!): Count!
        runComplete(input: RunCompleteInput!): Run!
        runCancel(input: RunCancelInput!): Run!
    }
`

const objectType = 'RunRoutine';
export const resolvers = {
    RunSortBy: RunSortBy,
    RunStatus: RunStatus,
    RunStepStatus: RunStepStatus,
    Query: {
        run: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Run>> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runs: async (_parent: undefined, { input }: IWrap<RunSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RunSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            const userData = assertRequestFrom(req, { isUser: true });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
        runsCount: async (_parent: undefined, { input }: IWrap<RunCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, objectType, prisma, req });
        },
    },
    Mutation: {
        runCreate: async (_parent: undefined, { input }: IWrap<RunCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Run>> => {
            await rateLimit({ info, maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runUpdate: async (_parent: undefined, { input }: IWrap<RunUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Run>> => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runDeleteAll: async (_parent: undefined, _input: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 25, req });
            return RunModel.danger.deleteAll(prisma, { __typename: 'User', id: userData.id });
        },
        runDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, maxUser: 100, req });
            return deleteManyHelper({ input, objectType, prisma, req });
        },
        runComplete: async (_parent: undefined, { input }: IWrap<RunCompleteInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Run>> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunModel.run.complete(prisma, userData, input, info);
        },
        runCancel: async (_parent: undefined, { input }: IWrap<RunCancelInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Run>> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return RunModel.run.cancel(prisma, userData, input, info);
        },
    }
}
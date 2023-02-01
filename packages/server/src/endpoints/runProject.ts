import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from '../types';
import { FindByIdInput, RunProjectSearchInput, RunProjectSortBy, RunProjectCreateInput, RunProjectUpdateInput, RunStatus, Count, RunProject, RunProjectCompleteInput, RunProjectCancelInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { assertRequestFrom } from '../auth';
import { RunProjectModel } from '../models';

export const typeDef = gql`
    enum RunProjectSortBy {
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

    input RunProjectCreateInput {
        id: ID!
        isPrivate: Boolean
        status: RunStatus!
        name: String!
        completedComplexity: Int
        contextSwitches: Int
        stepsCreate: [RunProjectStepCreateInput!]
        runProjectScheduleConnect: ID
        runProjectScheduleCreate: RunProjectScheduleCreateInput
        projectVersionConnect: ID!
        organizationId: ID
    }
    input RunProjectUpdateInput {
        id: ID!
        isPrivate: Boolean
        isStarted: Boolean # True if the run has started, and previously was scheduled
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunProjectStepCreateInput!]
        stepsUpdate: [RunProjectStepUpdateInput!]
        runProjectScheduleConnect: ID
        runProjectScheduleCreate: RunProjectScheduleCreateInput
    }
    type RunProject {
        id: ID!
        isPrivate: Boolean!
        completedComplexity: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        name: String!
        status: RunStatus!
        wasRunAutomaticaly: Boolean!
        projectVersion: ProjectVersion
        runProjectSchedule: RunProjectSchedule
        steps: [RunProjectStep!]!
        stepsCount: Int!
        user: User
        organization: Organization
        you: RunProjectYou!
    }

    type RunProjectYou {
        canDelete: Boolean!
        canEdit: Boolean!
        canView: Boolean!
    }

    input RunProjectSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        status: RunStatus
        projectVersionId: ID
        searchString: String
        sortBy: RunProjectSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type RunProjectSearchResult {
        pageInfo: PageInfo!
        edges: [RunProjectEdge!]!
    }

    type RunProjectEdge {
        cursor: String!
        node: RunProject!
    }

    input RunProjectCompleteInput {
        id: ID! # Run ID if "exists" is true, or routine version ID if "exists" is false
        completedComplexity: Int # Even though the runProject was completed, the user may not have completed every subroutine
        exists: Boolean # If true, runProject ID is provided, otherwise routine ID so we can create a runProject
        name: String # Title of routine, so runProject name stays consistent even if routine updates/deletes
        finalStepCreate: RunProjectStepCreateInput
        finalStepUpdate: RunProjectStepUpdateInput
        wasSuccessful: Boolean
    }
    input RunProjectCancelInput {
        id: ID!
    }

    extend type Query {
        runProject(input: FindByIdInput!): RunProject
        runProjects(input: RunProjectSearchInput!): RunProjectSearchResult!
    }

    extend type Mutation {
        runProjectCreate(input: RunProjectCreateInput!): RunProject!
        runProjectUpdate(input: RunProjectUpdateInput!): RunProject!
        runProjectDeleteAll: Count!
        runProjectComplete(input: RunProjectCompleteInput!): RunProject!
        runProjectCancel(input: RunProjectCancelInput!): RunProject!
    }
`

const objectType = 'RunProject';
export const resolvers: {
    RunProjectSortBy: typeof RunProjectSortBy;
    RunStatus: typeof RunStatus;
    Query: {
        runProject: GQLEndpoint<FindByIdInput, FindOneResult<RunProject>>;
        runProjects: GQLEndpoint<RunProjectSearchInput, FindManyResult<RunProject>>;
    },
    Mutation: {
        runProjectCreate: GQLEndpoint<RunProjectCreateInput, CreateOneResult<RunProject>>;
        runProjectUpdate: GQLEndpoint<RunProjectUpdateInput, UpdateOneResult<RunProject>>;
        runProjectDeleteAll: GQLEndpoint<{}, Count>;
        runProjectComplete: GQLEndpoint<RunProjectCompleteInput, RecursivePartial<RunProject>>;
        runProjectCancel: GQLEndpoint<RunProjectCancelInput, RecursivePartial<RunProject>>;
    }
} = {
    RunProjectSortBy,
    RunStatus,
    Query: {
        runProject: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        runProjects: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            const userData = assertRequestFrom(req, { isUser: true });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        runProjectCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        runProjectUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        runProjectDeleteAll: async (_p, _d, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 25, req });
            return (RunProjectModel as any).danger.deleteAll(prisma, { __typename: 'User', id: userData.id });
        },
        runProjectComplete: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return (RunProjectModel as any).run.complete(prisma, userData, input, info);
        },
        runProjectCancel: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return (RunProjectModel as any).run.cancel(prisma, userData, input, info);
        },
    }
}
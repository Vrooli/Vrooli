import { RunProjectSortBy, RunStatus } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsRunProject, RunProjectEndpoints } from "../logic/runProject";

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
        isPrivate: Boolean!
        completedComplexity: Int
        contextSwitches: Int
        name: String!
        status: RunStatus!
        timeElapsed: Int
        stepsCreate: [RunProjectStepCreateInput!]
        scheduleCreate: ScheduleCreateInput
        projectVersionConnect: ID!
        teamConnect: ID
    }
    input RunProjectUpdateInput {
        id: ID!
        isPrivate: Boolean
        completedComplexity: Int # Total completed complexity, including what was completed before this update
        contextSwitches: Int # Total contextSwitches, including what was completed before this update
        status: RunStatus
        timeElapsed: Int # Total time elapsed, including what was completed before this update
        stepsDelete: [ID!]
        stepsCreate: [RunProjectStepCreateInput!]
        stepsUpdate: [RunProjectStepUpdateInput!]
        scheduleCreate: ScheduleCreateInput
        scheduleUpdate: ScheduleUpdateInput
    }
    type RunProject {
        id: ID!
        isPrivate: Boolean!
        completedComplexity: Int!
        contextSwitches: Int!
        startedAt: Date
        timeElapsed: Int
        completedAt: Date
        lastStep: [Int!]
        name: String!
        status: RunStatus!
        projectVersion: ProjectVersion
        schedule: Schedule
        steps: [RunProjectStep!]!
        stepsCount: Int!
        team: Team
        user: User
        you: RunProjectYou!
    }

    type RunProjectYou {
        canDelete: Boolean!
        canUpdate: Boolean!
        canRead: Boolean!
    }

    input RunProjectSearchInput {
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

    extend type Query {
        runProject(input: FindByIdInput!): RunProject
        runProjects(input: RunProjectSearchInput!): RunProjectSearchResult!
    }

    extend type Mutation {
        runProjectCreate(input: RunProjectCreateInput!): RunProject!
        runProjectUpdate(input: RunProjectUpdateInput!): RunProject!
        runProjectDeleteAll: Count!
    }
`;

export const resolvers: {
    RunProjectSortBy: typeof RunProjectSortBy;
    RunStatus: typeof RunStatus;
    Query: EndpointsRunProject["Query"];
    Mutation: EndpointsRunProject["Mutation"];
} = {
    RunProjectSortBy,
    RunStatus,
    ...RunProjectEndpoints,
};

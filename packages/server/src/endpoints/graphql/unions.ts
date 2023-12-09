import { ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, RunProjectOrRunRoutineSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsUnions, UnionsEndpoints } from "../logic/unions";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum ProjectOrRoutineSortBy {
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IssuesAsc
        IssuesDesc
        PullRequestsAsc
        PullRequestsDesc
        QuestionsAsc
        QuestionsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        VersionsAsc
        VersionsDesc
        ViewsAsc
        ViewsDesc
    }

    enum ProjectOrOrganizationSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
    }

    enum RunProjectOrRunRoutineSortBy {
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

    union ProjectOrRoutine = Project | Routine
    union ProjectOrOrganization = Project | Organization
    union RunProjectOrRunRoutine = RunProject | RunRoutine

    input ProjectOrRoutineSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        hasCompleteVersion: Boolean
        hasCompleteVersionExceptions: [SearchException!]
        translationLanguagesLatestVersion: [String!]
        maxBookmarks: Int
        maxScore: Int
        minBookmarks: Int
        minScore: Int
        minViews: Int
        objectType: String
        organizationId: ID
        parentId: ID
        projectAfter: String
        reportId: ID
        resourceTypes: [ResourceUsedFor!]
        routineAfter: String
        routineIsInternal: Boolean
        routineMinComplexity: Int
        routineMaxComplexity: Int
        routineMinSimplicity: Int
        routineMaxSimplicity: Int
        routineMaxTimesCompleted: Int
        routineMinTimesCompleted: Int
        routineProjectId: ID
        searchString: String
        sortBy: ProjectOrRoutineSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }
    type ProjectOrRoutineSearchResult {
        pageInfo: ProjectOrRoutinePageInfo!
        edges: [ProjectOrRoutineEdge!]!
    }
    type ProjectOrRoutinePageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorRoutine: String
    }
    type ProjectOrRoutineEdge {
        cursor: String!
        node: ProjectOrRoutine!
    }

    input ProjectOrOrganizationSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        translationLanguagesLatestVersion: [String!]
        maxBookmarks: Int
        maxViews: Int
        minBookmarks: Int
        minViews: Int
        objectType: String
        organizationAfter: String
        organizationIsOpenToNewMembers: Boolean
        organizationProjectId: ID
        organizationRoutineId: ID
        projectAfter: String
        projectIsComplete: Boolean
        projectIsCompleteExceptions: [SearchException!]
        projectMaxScore: Int
        projectMinScore: Int
        projectOrganizationId: ID
        projectParentId: ID
        reportId: ID
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectOrOrganizationSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }
    type ProjectOrOrganizationSearchResult {
        pageInfo: ProjectOrOrganizationPageInfo!
        edges: [ProjectOrOrganizationEdge!]!
    }
    type ProjectOrOrganizationPageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorOrganization: String
    }
    type ProjectOrOrganizationEdge {
        cursor: String!
        node: ProjectOrOrganization!
    }

    input RunProjectOrRunRoutineSearchInput {
        createdTimeFrame: TimeFrame
        startedTimeFrame: TimeFrame
        completedTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        status: RunStatus
        statuses: [RunStatus!]
        objectType: String
        projectVersionId: ID
        routineVersionId: ID
        runProjectAfter: String
        runRoutineAfter: String
        searchString: String
        sortBy: RunProjectOrRunRoutineSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }
    type RunProjectOrRunRoutineSearchResult {
        pageInfo: RunProjectOrRunRoutinePageInfo!
        edges: [RunProjectOrRunRoutineEdge!]!
    }
    type RunProjectOrRunRoutinePageInfo {
        hasNextPage: Boolean!
        endCursorRunProject: String
        endCursorRunRoutine: String
    }
    type RunProjectOrRunRoutineEdge {
        cursor: String!
        node: RunProjectOrRunRoutine!
    }

    type Query {
        projectOrRoutines(input: ProjectOrRoutineSearchInput!): ProjectOrRoutineSearchResult!
        projectOrOrganizations(input: ProjectOrOrganizationSearchInput!): ProjectOrOrganizationSearchResult!
        runProjectOrRunRoutines(input: RunProjectOrRunRoutineSearchInput!): RunProjectOrRunRoutineSearchResult!
    }
`;

export const resolvers: {
    ProjectOrRoutineSortBy: typeof ProjectOrRoutineSortBy,
    ProjectOrOrganizationSortBy: typeof ProjectOrOrganizationSortBy,
    RunProjectOrRunRoutineSortBy: typeof RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: UnionResolver;
    ProjectOrOrganization: UnionResolver;
    Query: EndpointsUnions["Query"];
} = {
    ProjectOrRoutineSortBy,
    ProjectOrOrganizationSortBy,
    RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ProjectOrOrganization: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...UnionsEndpoints,
};

import { ProjectOrRoutineSortBy, ProjectOrTeamSortBy, RunProjectOrRunRoutineSortBy } from "@local/shared";
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

    enum ProjectOrTeamSortBy {
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
    union ProjectOrTeam = Project | Team
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
        teamId: ID
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

    input ProjectOrTeamSearchInput {
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        translationLanguagesLatestVersion: [String!]
        maxBookmarks: Int
        maxViews: Int
        minBookmarks: Int
        minViews: Int
        objectType: String
        projectAfter: String
        projectIsComplete: Boolean
        projectIsCompleteExceptions: [SearchException!]
        projectMaxScore: Int
        projectMinScore: Int
        projectParentId: ID
        projectTeamId: ID
        reportId: ID
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectOrTeamSortBy
        tags: [String!]
        teamAfter: String
        teamIsOpenToNewMembers: Boolean
        teamProjectId: ID
        teamRoutineId: ID
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }
    type ProjectOrTeamSearchResult {
        pageInfo: ProjectOrTeamPageInfo!
        edges: [ProjectOrTeamEdge!]!
    }
    type ProjectOrTeamPageInfo {
        hasNextPage: Boolean!
        endCursorProject: String
        endCursorTeam: String
    }
    type ProjectOrTeamEdge {
        cursor: String!
        node: ProjectOrTeam!
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
        runProjectOrRunRoutines(input: RunProjectOrRunRoutineSearchInput!): RunProjectOrRunRoutineSearchResult!
        projectOrTeams(input: ProjectOrTeamSearchInput!): ProjectOrTeamSearchResult!
    }
`;

export const resolvers: {
    ProjectOrRoutineSortBy: typeof ProjectOrRoutineSortBy,
    ProjectOrTeamSortBy: typeof ProjectOrTeamSortBy,
    RunProjectOrRunRoutineSortBy: typeof RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: UnionResolver;
    ProjectOrTeam: UnionResolver;
    Query: EndpointsUnions["Query"];
} = {
    ProjectOrRoutineSortBy,
    ProjectOrTeamSortBy,
    RunProjectOrRunRoutineSortBy,
    ProjectOrRoutine: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ProjectOrTeam: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...UnionsEndpoints,
};

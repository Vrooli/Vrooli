import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ProjectVersionSortBy, FindByIdInput, ProjectVersion, ProjectVersionSearchInput, ProjectVersionCreateInput, ProjectVersionUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ProjectVersionSortBy {
        CommentsAsc
        CommentsDesc
        ComplexityAsc
        ComplexityDesc
        DirectoryListingsAsc
        DirectoryListingsDesc
        ForksAsc
        ForksDesc
        DateCompletedAsc
        DateCompletedDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        RunProjectsAsc
        RunProjectsDesc
        SimplicityAsc
        SimplicityDesc
    }

    input ProjectVersionCreateInput {
        id: ID!
        isLatest: Boolean
        isPrivate: Boolean
        isComplete: Boolean
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        directoryListingsCreate: [ProjectVersionDirectoryCreateInput!]
        rootConnect: ID
        rootCreate: ProjectCreateInput
        suggestedNextByProjectConnect: [ID!]
        translationsCreate: [ProjectVersionTranslationCreateInput!]
    }
    input ProjectVersionUpdateInput {
        id: ID!
        isLatest: Boolean
        isPrivate: Boolean
        isComplete: Boolean
        versionIndex: Int
        versionLabel: String
        versionNotes: String
        translationsCreate: [ProjectVersionTranslationCreateInput!]
        translationsUpdate: [ProjectVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
        directoryListingsCreate: [ProjectVersionDirectoryCreateInput!]
        directoryListingsUpdate: [ProjectVersionDirectoryUpdateInput!]
        directoryListingsDelete: [ID!]
        suggestedNextByProjectConnect: [ID!]
        suggestedNextByProjectDisconnect: [ID!]
    }
    type ProjectVersion {
        id: ID!
        created_at: Date!
        updated_at: Date!
        isLatest: Boolean!
        isPrivate: Boolean!
        isComplete: Boolean!
        versionIndex: Int!
        versionLabel: String!
        versionNotes: String
        translations: [ProjectVersionTranslation!]!
        translationsCount: Int!
        directories: [ProjectVersionDirectory!]!
        directoriesCount: Int!
        directoryListings: [ProjectVersionDirectory!]!
        directoryListingsCount: Int!
        pullRequest: PullRequest
        reports: [Report!]!
        reportsCount: Int!
        root: Project!
        forks: [Project!]!
        forksCount: Int!
        comments: [Comment!]!
        commentsCount: Int!
        runsCount: Int!
        suggestedNextByProject: [Project!]!
        you: ProjectVersionYou!
    }

    type ProjectVersionYou {
        canComment: Boolean!
        canCopy: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canReport: Boolean!
        canUse: Boolean!
        canView: Boolean!
        runs: [RunProject!]!
    }

    input ProjectVersionTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input ProjectVersionTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ProjectVersionTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input ProjectVersionSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        directoryListingsId: ID
        ids: [ID!]
        minScoreRoot: Int
        minStarsRoot: Int
        minViewsRoot: Int
        createdById: ID
        ownedByUserId: ID
        ownedByOrganizationId: ID
        rootId: ID
        searchString: String
        sortBy: ProjectVersionSortBy
        tags: [String!]
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ProjectVersionSearchResult {
        pageInfo: PageInfo!
        edges: [ProjectVersionEdge!]!
    }

    type ProjectVersionEdge {
        cursor: String!
        node: ProjectVersion!
    }

    extend type Query {
        projectVersion(input: FindByIdInput!): ProjectVersion
        projectVersions(input: ProjectVersionSearchInput!): ProjectVersionSearchResult!
    }

    extend type Mutation {
        projectVersionCreate(input: ProjectVersionCreateInput!): ProjectVersion!
        projectVersionUpdate(input: ProjectVersionUpdateInput!): ProjectVersion!
    }
`

const objectType = 'ProjectVersion';
export const resolvers: {
    ProjectVersionSortBy: typeof ProjectVersionSortBy;
    Query: {
        projectVersion: GQLEndpoint<FindByIdInput, FindOneResult<ProjectVersion>>;
        projectVersions: GQLEndpoint<ProjectVersionSearchInput, FindManyResult<ProjectVersion>>;
    },
    Mutation: {
        projectVersionCreate: GQLEndpoint<ProjectVersionCreateInput, CreateOneResult<ProjectVersion>>;
        projectVersionUpdate: GQLEndpoint<ProjectVersionUpdateInput, UpdateOneResult<ProjectVersion>>;
    }
} = {
    ProjectVersionSortBy,
    Query: {
        projectVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        projectVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        projectVersionCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        projectVersionUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ProjectVersionSortBy, FindByIdInput, ProjectVersion, ProjectVersionSearchInput, ProjectVersionCreateInput, ProjectVersionUpdateInput } from './types';
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
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        parentId: ID
        resourceListsCreate: [ResourceListCreateInput!]
        rootId: ID!
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ProjectTranslationCreateInput!]
    }
    input ProjectVersionUpdateInput {
        id: ID!
        handle: String
        isComplete: Boolean
        isPrivate: Boolean
        organizationId: ID
        userId: ID
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [ProjectTranslationCreateInput!]
        translationsUpdate: [ProjectTranslationUpdateInput!]
    }
    type ProjectVersion {
        id: ID!
        completedAt: Date
        created_at: Date!
        updated_at: Date!
        handle: String
        isComplete: Boolean!
        isPrivate: Boolean!
        isStarred: Boolean!
        isUpvoted: Boolean
        isViewed: Boolean!
        score: Int!
        stars: Int!
        views: Int!
        comments: [Comment!]!
        commentsCount: Int!
        createdBy: User
        forks: [Project!]!
        owner: Owner
        parent: Project
        permissionsProject: ProjectPermission!
        reports: [Report!]!
        reportsCount: Int!
        resourceLists: [ResourceList!]
        routines: [Routine!]!
        starredBy: [User!]
        tags: [Tag!]!
        translations: [ProjectTranslation!]!
        wallets: [Wallet!]
    }

    type ProjectVersionPermission {
        canComment: Boolean!
        canDelete: Boolean!
        canEdit: Boolean!
        canStar: Boolean!
        canReport: Boolean!
        canView: Boolean!
        canVote: Boolean!
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
        ids: [ID!]
        isComplete: Boolean
        isCompleteExceptions: [SearchException!]
        languages: [String!]
        minScore: Int
        minStars: Int
        minViews: Int
        organizationId: ID
        parentId: ID
        reportId: ID
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        searchString: String
        sortBy: ProjectVersionSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
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
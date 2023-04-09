import { FindVersionInput, ProjectVersion, ProjectVersionContentsSearchInput, ProjectVersionContentsSearchResult, ProjectVersionContentsSortBy, ProjectVersionCreateInput, ProjectVersionSearchInput, ProjectVersionSortBy, ProjectVersionUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { CustomError } from '../events';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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

    # NOTE: This sort only applies to directories, not their items. We must order them on the client side, 
    # since Prisma only supports relationship ordering by _count
    enum ProjectVersionContentsSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input ProjectVersionCreateInput {
        id: ID!
        isPrivate: Boolean
        isComplete: Boolean
        versionLabel: String!
        versionNotes: String
        directoriesCreate: [ProjectVersionDirectoryCreateInput!]
        rootConnect: ID
        rootCreate: ProjectCreateInput
        suggestedNextByProjectConnect: [ID!]
        translationsCreate: [ProjectVersionTranslationCreateInput!]
    }
    input ProjectVersionUpdateInput {
        id: ID!
        isPrivate: Boolean
        isComplete: Boolean
        versionLabel: String
        versionNotes: String
        rootUpdate: ProjectUpdateInput
        translationsCreate: [ProjectVersionTranslationCreateInput!]
        translationsUpdate: [ProjectVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
        directoriesCreate: [ProjectVersionDirectoryCreateInput!]
        directoriesUpdate: [ProjectVersionDirectoryUpdateInput!]
        directoriesDelete: [ID!]
        suggestedNextByProjectConnect: [ID!]
        suggestedNextByProjectDisconnect: [ID!]
    }
    type ProjectVersion {
        id: ID!
        completedAt: Date
        complexity: Int!
        created_at: Date!
        updated_at: Date!
        isLatest: Boolean!
        isPrivate: Boolean!
        isComplete: Boolean!
        simplicity: Int!
        timesStarted: Int!
        timesCompleted: Int!
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
        runProjectsCount: Int!
        suggestedNextByProject: [Project!]!
        you: ProjectVersionYou!
    }

    type ProjectVersionYou {
        canComment: Boolean!
        canCopy: Boolean!
        canDelete: Boolean!
        canReport: Boolean!
        canUpdate: Boolean!
        canUse: Boolean!
        canRead: Boolean!
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
        isCompleteWithRoot: Boolean
        isCompleteWithRootExcludeOwnedByOrganizationId: ID
        isCompleteWithRootExcludeOwnedByUserId: ID
        isLatest: Boolean
        minComplexity: Int
        maxComplexity: Int
        minSimplicity: Int
        maxSimplicity: Int
        maxTimesCompleted: Int
        maxBookmarksRoot: Int
        maxScoreRoot: Int
        maxViewsRoot: Int
        minBookmarksRoot: Int
        minScoreRoot: Int
        minTimesCompleted: Int
        minViewsRoot: Int
        createdByIdRoot: ID
        ownedByUserIdRoot: ID
        ownedByOrganizationIdRoot: ID
        rootId: ID
        searchString: String
        sortBy: ProjectVersionSortBy
        tagsRoot: [String!]
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

    # NOTE: Search works different for directories than for other objects. 
    # Search edges can be a directory, or any of the items in a directory.
    input ProjectVersionContentsSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        directoryIds: [ID!] # Limit results to these directories
        projectVersionId: ID! # Limit results to one project version
        searchString: String
        sortBy: ProjectVersionContentsSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ProjectVersionContentsSearchResult {
        directory: ProjectVersionDirectory
        apiVersion: ApiVersion
        noteVersion: NoteVersion
        organization: Organization
        projectVersion: ProjectVersion
        routineVersion: RoutineVersion
        smartContractVersion: SmartContractVersion
        standardVersion: StandardVersion
    }

    extend type Query {
        projectVersion(input: FindVersionInput!): ProjectVersion
        projectVersions(input: ProjectVersionSearchInput!): ProjectVersionSearchResult!
        projectVersionContents(input: ProjectVersionContentsSearchInput!): ProjectVersionContentsSearchResult!
    }

    extend type Mutation {
        projectVersionCreate(input: ProjectVersionCreateInput!): ProjectVersion!
        projectVersionUpdate(input: ProjectVersionUpdateInput!): ProjectVersion!
    }
`

const objectType = 'ProjectVersion';
export const resolvers: {
    ProjectVersionSortBy: typeof ProjectVersionSortBy;
    ProjectVersionContentsSortBy: typeof ProjectVersionContentsSortBy;
    Query: {
        projectVersion: GQLEndpoint<FindVersionInput, FindOneResult<ProjectVersion>>;
        projectVersions: GQLEndpoint<ProjectVersionSearchInput, FindManyResult<ProjectVersion>>;
        projectVersionContents: GQLEndpoint<ProjectVersionContentsSearchInput, FindManyResult<ProjectVersionContentsSearchResult>>;
    },
    Mutation: {
        projectVersionCreate: GQLEndpoint<ProjectVersionCreateInput, CreateOneResult<ProjectVersion>>;
        projectVersionUpdate: GQLEndpoint<ProjectVersionUpdateInput, UpdateOneResult<ProjectVersion>>;
    }
} = {
    ProjectVersionSortBy,
    ProjectVersionContentsSortBy,
    Query: {
        projectVersion: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        projectVersions: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
        projectVersionContents: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            throw new CustomError('0000', 'NotImplemented', ['en'])
            // return ProjectVersionModel.query.searchContents(prisma, req, input, info);
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
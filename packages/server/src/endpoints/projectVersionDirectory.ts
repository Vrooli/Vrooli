import { gql } from 'apollo-server-express';
import { CustomError } from '../events';
import { ProjectVersionDirectoryModel } from '../models';
import { FindManyResult, GQLEndpoint } from '../types';
import { ProjectVersionDirectorySortBy } from '@shared/consts';
import { rateLimit } from '../middleware';

export const typeDef = gql`
    # NOTE: This sort only applies to directories, not their items. We must order them on the client side, 
    # since Prisma only supports relationship ordering by _count
    enum ProjectVersionDirectorySortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input ProjectVersionDirectoryCreateInput {
        id: ID!
        childOrder: String
        isRoot: Boolean
        parentDirectoryConnect: ID
        projectVersionConnect: ID!
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
    }
    input ProjectVersionDirectoryUpdateInput {
        id: ID!
        childOrder: String
        isRoot: Boolean
        parentDirectoryConnect: ID
        parentDirectoryDisconnect: ID
        projectVersionConnect: ID
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
        translationsUpdate: [ProjectVersionDirectoryTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type ProjectVersionDirectory {
        id: ID!
        created_at: Date!
        updated_at: Date!
        childOrder: String
        isRoot: Boolean!
        parentDirectory: ProjectVersionDirectory
        projectVersion: ProjectVersion
        children: [ProjectVersionDirectory!]!
        childApiVersions: [ApiVersion!]!
        childNoteVersions: [NoteVersion!]!
        childOrganizations: [Organization!]!
        childProjectVersions: [ProjectVersion!]!
        childRoutineVersions: [RoutineVersion!]!
        childSmartContractVersions: [SmartContractVersion!]!
        childStandardVersions: [StandardVersion!]!
        runProjectSteps: [RunProjectStep!]!
        translations: [ProjectVersionDirectoryTranslation!]!
    }

    input ProjectVersionDirectoryTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ProjectVersionDirectoryTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ProjectVersionDirectoryTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }

    # NOTE: Search works different for directories than for other objects. 
    # Search edges can be a directory, or any of the items in a directory.
    input ProjectVersionDirectorySearchInput {
        after: String
        createdTimeFrame: TimeFrame
        directoryIds: [ID!] # Limit results to these directories
        projectVersionId: ID! # Limit results to one project version
        searchString: String
        sortBy: ProjectVersionDirectorySortBy
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ProjectVersionDirectorySearchResult {
        pageInfo: PageInfo!
        edges: [ProjectVersionDirectoryEdge!]!
    }

    type ProjectVersionDirectoryEdge {
        cursor: String!
        node: ProjectVersionDirectorySearchItem!
    }

    type ProjectVersionDirectorySearchItem {
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
        projectVersionDirectories(input: ProjectVersionDirectorySearchInput!): ProjectVersionDirectorySearchResult!
    }
`

const objectType = 'ProjectVersionDirectory';
export const resolvers: {
    ProjectVersionDirectorySortBy: typeof ProjectVersionDirectorySortBy;
    Query: {
        projectVersionDirectories: GQLEndpoint<any, FindManyResult<any>>;
    },
} = {
    ProjectVersionDirectorySortBy,
    Query: {
        projectVersionDirectories: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            throw new CustomError('0000', 'NotImplemented', ['en'])
            // return ProjectVersionDirectoryModel.query.searchNested(prisma, req, input, info);
        },
    },
}
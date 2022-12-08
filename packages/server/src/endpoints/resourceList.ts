import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, ResourceList, ResourceListCreateInput, ResourceListUpdateInput, ResourceListSortBy, ResourceListUsedFor, ResourceListSearchInput } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ResourceListSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IndexAsc
        IndexDesc
    }

    enum ResourceListUsedFor {
        Custom
        Display
        Learn
        Research
        Develop
    }

    input ResourceListCreateInput {
        id: ID!
        index: Int
        usedFor: ResourceListUsedFor!
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        translationsCreate: [ResourceListTranslationCreateInput!]
        resourcesCreate: [ResourceCreateInput!]
    }
    input ResourceListUpdateInput {
        id: ID!
        index: Int
        usedFor: ResourceListUsedFor
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        translationsDelete: [ID!]
        translationsCreate: [ResourceListTranslationCreateInput!]
        translationsUpdate: [ResourceListTranslationUpdateInput!]
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
    }
    type ResourceList {
        id: ID!
        created_at: Date!
        updated_at: Date!
        index: Int
        usedFor: ResourceListUsedFor
        # api: Api
        organization: Organization
        # post: Post
        project: Project
        routine: Routine
        # smartContract: SmartContract
        standard: Standard
        # userSchedule: UserSchedule
        translations: [ResourceListTranslation!]!
        resources: [Resource!]!
    }

    input ResourceListTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ResourceListTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ResourceListTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }

    input ResourceListSearchInput {
        organizationId: ID
        projectId: ID
        routineId: ID
        userId: ID
        ids: [ID!]
        languages: [String!]
        sortBy: ResourceListSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    type ResourceListSearchResult {
        pageInfo: PageInfo!
        edges: [ResourceListEdge!]!
    }

    type ResourceListEdge {
        cursor: String!
        node: ResourceList!
    }

    extend type Query {
        resourceList(input: FindByIdInput!): Resource
        resourceLists(input: ResourceListSearchInput!): ResourceListSearchResult!
    }

    extend type Mutation {
        resourceListCreate(input: ResourceListCreateInput!): ResourceList!
        resourceListUpdate(input: ResourceListUpdateInput!): ResourceList!
    }
`

const objectType = 'ResourceList';
export const resolvers: {
    ResourceListSortBy: typeof ResourceListSortBy;
    ResourceListUsedFor: typeof ResourceListUsedFor;
    Query: {
        resourceList: GQLEndpoint<FindByIdInput, FindOneResult<ResourceList>>;
        resourceLists: GQLEndpoint<ResourceListSearchInput, FindManyResult<ResourceList>>;
    },
    Mutation: {
        resourceListCreate: GQLEndpoint<ResourceListCreateInput, CreateOneResult<ResourceList>>;
        resourceListUpdate: GQLEndpoint<ResourceListUpdateInput, UpdateOneResult<ResourceList>>;
    }
} = {
    ResourceListSortBy: ResourceListSortBy,
    ResourceListUsedFor: ResourceListUsedFor,
    Query: {
        resourceList: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        resourceLists: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        resourceListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        resourceListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    }
}
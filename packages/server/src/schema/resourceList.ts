import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ResourceList, ResourceListCountInput, ResourceListCreateInput, ResourceListUpdateInput, ResourceListSearchResult, ResourceListSortBy, ResourceListUsedFor, ResourceListSearchInput } from './types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { ResourceListModel } from '../models';

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
        title: String
    }
    input ResourceListTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type ResourceListTranslation {
        id: ID!
        language: String!
        description: String
        title: String
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

    # Return type for search result
    type ResourceListSearchResult {
        pageInfo: PageInfo!
        edges: [ResourceListEdge!]!
    }

    # Return type for search result edge
    type ResourceListEdge {
        cursor: String!
        node: ResourceList!
    }

    # Input for count
    input ResourceListCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        resourceList(input: FindByIdInput!): Resource
        resourceLists(input: ResourceListSearchInput!): ResourceListSearchResult!
        resourceListsCount(input: ResourceListCountInput!): Int!
    }

    extend type Mutation {
        resourceListCreate(input: ResourceListCreateInput!): ResourceList!
        resourceListUpdate(input: ResourceListUpdateInput!): ResourceList!
        resourceListDeleteMany(input: DeleteManyInput!): Count!
    }
`

const objectType = 'ResourceList';
export const resolvers = {
    ResourceListSortBy: ResourceListSortBy,
    ResourceListUsedFor: ResourceListUsedFor,
    Query: {
        resourceList: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList> | null> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        resourceLists: async (_parent: undefined, { input }: IWrap<ResourceListSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ResourceListSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
        resourceListsCount: async (_parent: undefined, { input }: IWrap<ResourceListCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, objectType, prisma, req });
        },
    },
    Mutation: {
        resourceListCreate: async (_parent: undefined, { input }: IWrap<ResourceListCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList>> => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        resourceListUpdate: async (_parent: undefined, { input }: IWrap<ResourceListUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList>> => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        resourceListDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, maxUser: 100, req });
            return deleteManyHelper({ input, objectType, prisma, req });
        },
    }
}
import { gql } from 'apollo-server-express';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, updateHelper, ResourceListModel } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, ResourceList, ResourceListCountInput, ResourceListCreateInput, ResourceListUpdateInput, ResourceListSearchResult, ResourceListSortBy, ResourceListUsedFor, ResourceListSearchInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

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
        organization: Organization
        project: Project
        routine: Routine
        user: User
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

export const resolvers = {
    ResourceListSortBy: ResourceListSortBy,
    ResourceListUsedFor: ResourceListUsedFor,
    Query: {
        resourceList: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList> | null> => {
            await rateLimit({ info, max: 1000, req });
            return readOneHelper({ info, input, model: ResourceListModel, prisma, userId: req.userId });
        },
        resourceLists: async (_parent: undefined, { input }: IWrap<ResourceListSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ResourceListSearchResult> => {
            await rateLimit({ info, max: 1000, req });
            return readManyHelper({ info, input, model: ResourceListModel, prisma, userId: req.userId });
        },
        resourceListsCount: async (_parent: undefined, { input }: IWrap<ResourceListCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, max: 1000, req });
            return countHelper({ input, model: ResourceListModel, prisma });
        },
    },
    Mutation: {
        resourceListCreate: async (_parent: undefined, { input }: IWrap<ResourceListCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList>> => {
            await rateLimit({ info, max: 100, byAccountOrKey: true, req });
            return createHelper({ info, input, model: ResourceListModel, prisma, userId: req.userId });
        },
        resourceListUpdate: async (_parent: undefined, { input }: IWrap<ResourceListUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<ResourceList>> => {
            await rateLimit({ info, max: 250, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: ResourceListModel, prisma, userId: req.userId });
        },
        resourceListDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, max: 100, byAccountOrKey: true, req });
            return deleteManyHelper({ input, model: ResourceListModel, prisma, userId: req.userId });
        },
    }
}
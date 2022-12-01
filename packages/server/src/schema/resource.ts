import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteManyInput, FindByIdInput, Resource, ResourceCountInput, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSearchResult, ResourceFor, ResourceSortBy, ResourceUsedFor } from './types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, deleteManyHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { ResourceModel } from '../models';

export const typeDef = gql`
    enum ResourceFor {
        Organization
        Project
        Routine
        User
    }

    enum ResourceSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IndexAsc
        IndexDesc
    }

    enum ResourceUsedFor {
        Community
        Context
        Developer
        Donation
        ExternalService
        Feed
        Install
        Learning
        Notes
        OfficialWebsite
        Proposal
        Related
        Researching
        Scheduling
        Social
        Tutorial
    }

    input ResourceCreateInput {
        id: ID!
        listId: ID!
        index: Int
        link: String!
        translationsCreate: [ResourceTranslationCreateInput!]
        usedFor: ResourceUsedFor!
    }
    input ResourceUpdateInput {
        id: ID!
        listId: ID
        index: Int
        link: String
        translationsDelete: [ID!]
        translationsCreate: [ResourceTranslationCreateInput!]
        translationsUpdate: [ResourceTranslationUpdateInput!]
        usedFor: ResourceUsedFor
    }
    type Resource {
        id: ID!
        created_at: Date!
        updated_at: Date!
        listId: ID!
        index: Int
        link: String!
        translations: [ResourceTranslation!]!
        usedFor: ResourceUsedFor
    }

    input ResourceTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String
    }
    input ResourceTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type ResourceTranslation {
        id: ID!
        language: String!
        description: String
        title: String
    }

    input ResourceSearchInput {
        forId: ID
        forType: ResourceFor
        ids: [ID!]
        languages: [String!]
        sortBy: ResourceSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type ResourceSearchResult {
        pageInfo: PageInfo!
        edges: [ResourceEdge!]!
    }

    # Return type for search result edge
    type ResourceEdge {
        cursor: String!
        node: Resource!
    }

    # Input for count
    input ResourceCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        resource(input: FindByIdInput!): Resource
        resources(input: ResourceSearchInput!): ResourceSearchResult!
        resourcesCount(input: ResourceCountInput!): Int!
    }

    extend type Mutation {
        resourceCreate(input: ResourceCreateInput!): Resource!
        resourceUpdate(input: ResourceUpdateInput!): Resource!
        resourceDeleteMany(input: DeleteManyInput!): Count!
    }
`

const objectType = 'Resource';
export const resolvers = {
    ResourceFor: ResourceFor,
    ResourceSortBy: ResourceSortBy,
    ResourceUsedFor: ResourceUsedFor,
    Query: {
        resource: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource> | null> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        resources: async (_parent: undefined, { input }: IWrap<ResourceSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ResourceSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
        resourcesCount: async (_parent: undefined, { input }: IWrap<ResourceCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, objectType, prisma, req })
        },
    },
    Mutation: {
        resourceCreate: async (_parent: undefined, { input }: IWrap<ResourceCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        resourceUpdate: async (_parent: undefined, { input }: IWrap<ResourceUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Resource>> => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        resourceDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Count> => {
            await rateLimit({ info, maxUser: 500, req });
            return deleteManyHelper({ input, objectType, prisma, req })
        },
    }
}
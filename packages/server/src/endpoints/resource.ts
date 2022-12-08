import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Resource, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceFor, ResourceSortBy, ResourceUsedFor } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
        UsedForAsc
        UsedForDesc
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
        name: String
    }
    input ResourceTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ResourceTranslation {
        id: ID!
        language: String!
        description: String
        name: String
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

    type ResourceSearchResult {
        pageInfo: PageInfo!
        edges: [ResourceEdge!]!
    }

    type ResourceEdge {
        cursor: String!
        node: Resource!
    }

    extend type Query {
        resource(input: FindByIdInput!): Resource
        resources(input: ResourceSearchInput!): ResourceSearchResult!
    }

    extend type Mutation {
        resourceCreate(input: ResourceCreateInput!): Resource!
        resourceUpdate(input: ResourceUpdateInput!): Resource!
    }
`

const objectType = 'Resource';
export const resolvers: {
    ResourceFor: typeof ResourceFor;
    ResourceSortBy: typeof ResourceSortBy;
    ResourceUsedFor: typeof ResourceUsedFor;
    Query: {
        resource: GQLEndpoint<FindByIdInput, FindOneResult<Resource>>;
        resources: GQLEndpoint<ResourceSearchInput, FindManyResult<Resource>>;
    },
    Mutation: {
        resourceCreate: GQLEndpoint<ResourceCreateInput, CreateOneResult<Resource>>;
        resourceUpdate: GQLEndpoint<ResourceUpdateInput, UpdateOneResult<Resource>>;
    }
} = {
    ResourceFor: ResourceFor,
    ResourceSortBy: ResourceSortBy,
    ResourceUsedFor: ResourceUsedFor,
    Query: {
        resource: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        resources: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        resourceCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        resourceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
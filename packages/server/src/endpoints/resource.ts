import { FindByIdInput, Resource, ResourceCreateInput, ResourceSearchInput, ResourceSortBy, ResourceUpdateInput, ResourceUsedFor } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
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
        index: Int
        link: String!
        usedFor: ResourceUsedFor!
        listConnect: ID!
        translationsCreate: [ResourceTranslationCreateInput!]
    }
    input ResourceUpdateInput {
        id: ID!
        index: Int
        link: String
        usedFor: ResourceUsedFor
        listConnect: ID
        translationsDelete: [ID!]
        translationsCreate: [ResourceTranslationCreateInput!]
        translationsUpdate: [ResourceTranslationUpdateInput!]
    }
    type Resource {
        id: ID!
        created_at: Date!
        updated_at: Date!
        index: Int
        link: String!
        usedFor: ResourceUsedFor!
        list: ResourceList!
        translations: [ResourceTranslation!]!
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
        ids: [ID!]
        sortBy: ResourceSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        resourceListId: ID
        searchString: String
        after: String
        take: Int
        translationLanguages: [String!]
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
`;

const objectType = "Resource";
export const resolvers: {
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
    ResourceSortBy,
    ResourceUsedFor,
    Query: {
        resource: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        resources: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        resourceCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        resourceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

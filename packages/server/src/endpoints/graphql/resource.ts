import { ResourceSortBy, ResourceUsedFor } from "@local/shared";
import { EndpointsResource, ResourceEndpoints } from "../logic/resource";

export const typeDef = `#graphql
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
        listConnect: ID
        listCreate: ResourceListCreateInput
        translationsCreate: [ResourceTranslationCreateInput!]
    }
    input ResourceUpdateInput {
        id: ID!
        index: Int
        link: String
        usedFor: ResourceUsedFor
        listConnect: ID
        listCreate: ResourceListCreateInput
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
        language: String!
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

export const resolvers: {
    ResourceSortBy: typeof ResourceSortBy;
    ResourceUsedFor: typeof ResourceUsedFor;
    Query: EndpointsResource["Query"];
    Mutation: EndpointsResource["Mutation"];
} = {
    ResourceSortBy,
    ResourceUsedFor,
    ...ResourceEndpoints,
};

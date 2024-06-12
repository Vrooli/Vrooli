import { ResourceListFor, ResourceListSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsResourceList, ResourceListEndpoints } from "../logic/resourceList";

export const typeDef = gql`
    enum ResourceListSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IndexAsc
        IndexDesc
    }

    enum ResourceListFor {
        ApiVersion
        CodeVersion
        FocusMode
        Post
        ProjectVersion
        RoutineVersion
        StandardVersion
        Team
    }   

    union ResourceListOn = ApiVersion | CodeVersion | FocusMode | Post | ProjectVersion | RoutineVersion | StandardVersion | Team

    input ResourceListCreateInput {
        id: ID!
        listForType: ResourceListFor!
        listForConnect: ID!
        resourcesCreate: [ResourceCreateInput!]
        translationsCreate: [ResourceListTranslationCreateInput!]
    }
    input ResourceListUpdateInput {
        id: ID!
        resourcesDelete: [ID!]
        resourcesCreate: [ResourceCreateInput!]
        resourcesUpdate: [ResourceUpdateInput!]
        translationsDelete: [ID!]
        translationsCreate: [ResourceListTranslationCreateInput!]
        translationsUpdate: [ResourceListTranslationUpdateInput!]
    }
    type ResourceList {
        id: ID!
        created_at: Date!
        updated_at: Date!
        listFor: ResourceListOn!
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
        language: String!
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
        ids: [ID!]
        sortBy: ResourceListSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
        apiVersionId: ID
        codeVersionId: ID
        focusModeId: ID
        postId: ID
        projectVersionId: ID
        routineVersionId: ID
        standardVersionId: ID
        teamId: ID
        translationLanguages: [String!]
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
`;

export const resolvers: {
    ResourceListFor: typeof ResourceListFor;
    ResourceListSortBy: typeof ResourceListSortBy;
    Query: EndpointsResourceList["Query"];
    Mutation: EndpointsResourceList["Mutation"];
} = {
    ResourceListFor,
    ResourceListSortBy,
    ...ResourceListEndpoints,
};

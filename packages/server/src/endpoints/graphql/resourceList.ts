import { ResourceListSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsResourceList, ResourceListEndpoints } from "../logic";

export const typeDef = gql`
    enum ResourceListSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        IndexAsc
        IndexDesc
    }

    input ResourceListCreateInput {
        id: ID!
        apiVersionConnect: ID
        focusModeConnect: ID
        organizationConnect: ID
        postConnect: ID
        projectVersionConnect: ID
        routineVersionConnect: ID
        smartContractVersionConnect: ID
        standardVersionConnect: ID
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
        apiVersion: ApiVersion
        organization: Organization
        post: Post
        projectVersion: ProjectVersion
        routineVersion: RoutineVersion
        smartContractVersion: SmartContractVersion
        standardVersion: StandardVersion
        focusMode: FocusMode
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
        organizationId: ID
        postId: ID
        projectVersionId: ID
        routineVersionId: ID
        smartContractVersionId: ID
        standardVersionId: ID
        translationLanguages: [String!]
        focusModeId: ID
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
    ResourceListSortBy: typeof ResourceListSortBy;
    Query: EndpointsResourceList["Query"];
    Mutation: EndpointsResourceList["Mutation"];
} = {
    ResourceListSortBy,
    ...ResourceListEndpoints,
};
import { FindByIdInput, ResourceList, ResourceListCreateInput, ResourceListSearchInput, ResourceListSortBy, ResourceListUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

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

const objectType = "ResourceList";
export const resolvers: {
    ResourceListSortBy: typeof ResourceListSortBy;
    Query: {
        resourceList: GQLEndpoint<FindByIdInput, FindOneResult<ResourceList>>;
        resourceLists: GQLEndpoint<ResourceListSearchInput, FindManyResult<ResourceList>>;
    },
    Mutation: {
        resourceListCreate: GQLEndpoint<ResourceListCreateInput, CreateOneResult<ResourceList>>;
        resourceListUpdate: GQLEndpoint<ResourceListUpdateInput, UpdateOneResult<ResourceList>>;
    }
} = {
    ResourceListSortBy,
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
    },
};

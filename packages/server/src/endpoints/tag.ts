import { FindByIdInput, Tag, TagCreateInput, TagSearchInput, TagSortBy, TagUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readOneHelper, updateHelper } from "../actions";
import { readManyWithEmbeddingsHelper } from "../actions/readManyWithEmbeddingsHelper";
import { rateLimit } from "../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../types";

export const typeDef = gql`
    enum TagSortBy {
        New
        Top
    }

    input TagCreateInput {
        anonymous: Boolean
        tag: String!
        translationsCreate: [TagTranslationCreateInput!]
    }
    input TagUpdateInput {
        anonymous: Boolean
        tag: String!
        translationsDelete: [ID!]
        translationsCreate: [TagTranslationCreateInput!]
        translationsUpdate: [TagTranslationUpdateInput!]
    }

    type Tag {
        id: ID!
        tag: String!
        created_at: Date!
        updated_at: Date!
        bookmarks: Int!
        bookmarkedBy: [User!]!
        translations: [TagTranslation!]!
        apis: [Api!]!
        notes: [Note!]!
        organizations: [Organization!]!
        posts: [Post!]!
        projects: [Project!]!
        reports: [Report!]!
        routines: [Routine!]!
        smartContracts: [SmartContract!]!
        standards: [Standard!]!
        you: TagYou!
    }

    type TagYou {
        isOwn: Boolean!
        isBookmarked: Boolean!
    }

    input TagTranslationCreateInput {
        id: ID!
        language: String!
        description: String
    }
    input TagTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
    }
    type TagTranslation {
        id: ID!
        language: String!
        description: String
    }

    input TagSearchInput {
        after: String
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        maxBookmarks: Int
        minBookmarks: Int
        searchString: String
        sortBy: TagSortBy
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
    }

    type TagSearchResult {
        pageInfo: PageInfo!
        edges: [TagEdge!]!
    }

    type TagEdge {
        cursor: String!
        node: Tag!
    }

    extend type Query {
        tag(input: FindByIdInput!): Tag
        tags(input: TagSearchInput!): TagSearchResult!
    }

    extend type Mutation {
        tagCreate(input: TagCreateInput!): Tag!
        tagUpdate(input: TagUpdateInput!): Tag!
    }
`;

const objectType = "Tag";
export const resolvers: {
    TagSortBy: typeof TagSortBy;
    Query: {
        tag: GQLEndpoint<FindByIdInput, FindOneResult<Tag>>;
        tags: GQLEndpoint<TagSearchInput, FindManyResult<Tag>>;
    },
    Mutation: {
        tagCreate: GQLEndpoint<TagCreateInput, CreateOneResult<Tag>>;
        tagUpdate: GQLEndpoint<TagUpdateInput, UpdateOneResult<Tag>>;
    }
} = {
    TagSortBy,
    Query: {
        tag: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        tags: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyWithEmbeddingsHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        tagCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        tagUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

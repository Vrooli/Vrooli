import { TagSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsTag, TagEndpoints } from "../logic";

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

export const resolvers: {
    TagSortBy: typeof TagSortBy;
    Query: EndpointsTag["Query"];
    Mutation: EndpointsTag["Mutation"];
} = {
    TagSortBy,
    ...TagEndpoints,
};
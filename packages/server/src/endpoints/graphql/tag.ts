import { TagSortBy } from "@local/shared";
import { EndpointsTag, TagEndpoints } from "../logic/tag";

export const typeDef = `#graphql
    enum TagSortBy {
        EmbedDateCreatedAsc
        EmbedDateCreatedDesc
        EmbedDateUpdatedAsc
        EmbedDateUpdatedDesc
        EmbedTopAsc
        EmbedTopDesc
    }

    input TagCreateInput {
        id: ID!
        anonymous: Boolean
        tag: String!
        translationsCreate: [TagTranslationCreateInput!]
    }
    input TagUpdateInput {
        id: ID!
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
        codes: [Code!]!
        notes: [Note!]!
        posts: [Post!]!
        projects: [Project!]!
        reports: [Report!]!
        routines: [Routine!]!
        standards: [Standard!]!
        teams: [Team!]!
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
        after: String # Used when there's no search string, for cursor-based pagination
        createdById: ID
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        ids: [ID!]
        maxBookmarks: Int
        minBookmarks: Int
        offset: Int # Used whent there is a search string, since we have to calculate a score for each result (meaning it's not ordered)
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

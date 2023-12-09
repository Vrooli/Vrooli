import { PostSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsPost, PostEndpoints } from "../logic/post";

export const typeDef = gql`
    enum PostSortBy {
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        ReportsAsc
        ReportsDesc
        RepostsAsc
        RepostsDesc
        ScoreAsc
        ScoreDesc
        BookmarksAsc
        BookmarksDesc
        ViewsAsc
        ViewsDesc
    }

    input PostCreateInput {
        id: ID!
        isPinned: Boolean
        isPrivate: Boolean!
        organizationConnect: ID
        repostedFromConnect: ID
        resourceListCreate: ResourceListCreateInput
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        translationsCreate: [ApiVersionTranslationCreateInput!]
        userConnect: ID
    }
    input PostUpdateInput {
        id: ID!
        isPinned: Boolean
        isPrivate: Boolean
        resourceListUpdate: ResourceListUpdateInput
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
        tagsDisconnect: [String!]
        translationsCreate: [ApiVersionTranslationCreateInput!]
        translationsUpdate: [ApiVersionTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type Post {
        id: ID!
        created_at: Date!
        updated_at: Date!
        comments: [Comment!]!
        commentsCount: Int!
        owner: Owner!
        reports: [Report!]!
        repostedFrom: Post
        reposts: [Post!]!
        repostsCount: Int!
        resourceList: ResourceList!
        score: Int!
        bookmarks: Int!
        views: Int!
        bookmarkedBy: [User!]!
        tags: [Tag!]!
        translations: [PostTranslation!]!
    }

    input PostTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input PostTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    type PostTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }

    input PostSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        excludeIds: [ID!]
        isPinned: Boolean
        ids: [ID!]
        translationLanguages: [String!]
        maxScore: Int
        maxBookmarks: Int
        minScore: Int
        minBookmarks: Int
        organizationId: ID
        userId: ID
        repostedFromIds: [ID!]
        searchString: String
        sortBy: PostSortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type PostSearchResult {
        pageInfo: PageInfo!
        edges: [PostEdge!]!
    }

    type PostEdge {
        cursor: String!
        node: Post!
    }

    extend type Query {
        post(input: FindByIdInput!): Post
        posts(input: PostSearchInput!): PostSearchResult!
    }

    extend type Mutation {
        postCreate(input: PostCreateInput!): Post!
        postUpdate(input: PostUpdateInput!): Post!
    }
`;

export const resolvers: {
    PostSortBy: typeof PostSortBy;
    Query: EndpointsPost["Query"];
    Mutation: EndpointsPost["Mutation"];
} = {
    PostSortBy,
    ...PostEndpoints,
};

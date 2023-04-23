import { PostSortBy } from "@local/consts";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../actions";
import { rateLimit } from "../middleware";
export const typeDef = gql `
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
        isPrivate: Boolean
        organizationConnect: ID
        repostedFromConnect: ID
        resourceListCreate: ResourceListCreateInput
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
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
        language: String
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
const objectType = "Post";
export const resolvers = {
    PostSortBy,
    Query: {
        post: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        posts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        postUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};
//# sourceMappingURL=post.js.map
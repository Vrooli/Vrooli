import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Post, PostSearchInput, PostCreateInput, PostUpdateInput, PostSortBy } from './types';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

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
        StarsAsc
        StarsDesc
        ViewsAsc
        ViewsDesc
    }

    input PostCreateInput {
        id: ID!
        isPinned: Boolean
        isPublic: Boolean
        organizationConnect: ID
        repostedFromConnect: ID
        resourceListCreate: ResourceListCreateInput
        tagsConnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    input PostUpdateInput {
        id: ID!
        isPinned: Boolean
        isPublic: Boolean
        resourceListUpdate: ResourceListUpdateInput
        tagsConnect: [String!]
        tagsDisconnect: [String!]
        tagsCreate: [TagCreateInput!]
    }
    type Post {
        id: ID!
        created_at: Date!
        updated_at: Date!
        comments: [Comment!]!
        owner: Owner!
        reports: [Report!]!
        repostedFrom: Post
        reposts: [Post!]!
        score: Int!
        stars: Int!
        views: Int!
        starredBy: [User!]!
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
        languages: [String!]
        minScore: Int
        minStars: Int
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
`

const objectType = 'Post';
export const resolvers: {
    PostSortBy: typeof PostSortBy;
    Query: {
        post: GQLEndpoint<FindByIdInput, FindOneResult<Post>>;
        posts: GQLEndpoint<PostSearchInput, FindManyResult<Post>>;
    },
    Mutation: {
        postCreate: GQLEndpoint<PostCreateInput, CreateOneResult<Post>>;
        postUpdate: GQLEndpoint<PostUpdateInput, UpdateOneResult<Post>>;
    }
} = {
    PostSortBy,
    Query: {
        post: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        posts: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        postCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        postUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
import { gql } from 'apollo-server-express';
import { BookmarkCreateInput, BookmarkSortBy, BookmarkUpdateInput, FindByIdInput } from '@shared/consts';
import { Bookmark, BookmarkFor, BookmarkSearchInput } from '@shared/consts';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { rateLimit } from '../middleware';
import { resolveUnion } from './resolvers';
import { assertRequestFrom } from '../auth/request';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum BookmarkSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }
    
    enum BookmarkFor {
        Api
        Comment
        Issue
        Note
        Organization
        Post
        Project
        Question
        QuestionAnswer
        Quiz
        Routine
        SmartContract
        Standard
        Tag
        User
    }   

    union BookmarkTo = Api | Comment | Issue | Note | Organization | Project | Post | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard | Tag | User

    input BookmarkCreateInput {
        id: ID!
        bookmarkFor: BookmarkFor!
        label: String
        forConnect: ID!
    }
    input BookmarkUpdateInput {
        id: ID!
        label: String
    }
    type Bookmark {
        id: ID!
        created_at: Date!
        updated_at: Date!
        label: String!
        by: User!
        to: BookmarkTo!
    }

    input BookmarkSearchInput {
        after: String
        excludeLinkedToTag: Boolean
        ids: [ID!]
        searchString: String
        sortBy: BookmarkSortBy
        take: Int
    }
    type BookmarkSearchResult {
        pageInfo: PageInfo!
        edges: [BookmarkEdge!]!
    }
    type BookmarkEdge {
        cursor: String!
        node: Bookmark!
    }

    extend type Query {
        bookmark(input: FindByIdInput!): Bookmark
        bookmarks(input: BookmarkSearchInput!): BookmarkSearchResult!
    }

    extend type Mutation {
        bookmarkCreate(input: BookmarkCreateInput!): Bookmark!
        bookmarkUpdate(input: BookmarkUpdateInput!): Bookmark!
    }
`

const objectType = 'Bookmark';
export const resolvers: {
    BookmarkSortBy: typeof BookmarkSortBy;
    BookmarkFor: typeof BookmarkFor;
    BookmarkTo: UnionResolver;
    Query: {
        bookmark: GQLEndpoint<FindByIdInput, FindOneResult<Bookmark>>;
        bookmarks: GQLEndpoint<BookmarkSearchInput, FindManyResult<Bookmark>>;
    },
    Mutation: {
        bookmarkCreate: GQLEndpoint<BookmarkCreateInput, CreateOneResult<Bookmark>>;
        bookmarkUpdate: GQLEndpoint<BookmarkUpdateInput, UpdateOneResult<Bookmark>>;
    }
} = {
    BookmarkSortBy,
    BookmarkFor,
    BookmarkTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        bookmark: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        bookmarks: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        bookmarkCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        bookmarkUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
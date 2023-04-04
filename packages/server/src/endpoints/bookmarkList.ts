import { BookmarkList, BookmarkListCreateInput, BookmarkListSearchInput, BookmarkListSortBy, BookmarkListUpdateInput, FindByIdInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { assertRequestFrom } from '../auth/request';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

export const typeDef = gql`
    enum BookmarkListSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
        IndexAsc
        IndexDesc
        LabelAsc
        LabelDesc
    }

    input BookmarkListCreateInput {
        id: ID!
        label: String!
        bookmarksConnect: [ID!]
        bookmarksCreate: [BookmarkCreateInput!]
    }
    input BookmarkListUpdateInput {
        id: ID!
        label: String
        bookmarksConnect: [ID!]
        bookmarksCreate: [BookmarkCreateInput!]
        bookmarksUpdate: [BookmarkUpdateInput!]
        bookmarksDelete: [ID!]
    }
    type BookmarkList {
        id: ID!
        created_at: Date!
        updated_at: Date!
        label: String!
        bookmarks: [Bookmark!]!
        bookmarksCount: Int!
    }

    input BookmarkListSearchInput {
        after: String
        ids: [ID!]
        labelsIds: [String!]
        searchString: String
        sortBy: BookmarkListSortBy
        take: Int
    }
    type BookmarkListSearchResult {
        pageInfo: PageInfo!
        edges: [BookmarkListEdge!]!
    }
    type BookmarkListEdge {
        cursor: String!
        node: BookmarkList!
    }

    extend type Query {
        bookmarkList(input: FindByIdInput!): BookmarkList
        bookmarkLists(input: BookmarkListSearchInput!): BookmarkListSearchResult!
    }

    extend type Mutation {
        bookmarkListCreate(input: BookmarkListCreateInput!): BookmarkList!
        bookmarkListUpdate(input: BookmarkListUpdateInput!): BookmarkList!
    }
`

const objectType = 'BookmarkList';
export const resolvers: {
    BookmarkListSortBy: typeof BookmarkListSortBy;
    Query: {
        bookmarkList: GQLEndpoint<FindByIdInput, FindOneResult<BookmarkList>>;
        bookmarkLists: GQLEndpoint<BookmarkListSearchInput, FindManyResult<BookmarkList>>;
    },
    Mutation: {
        bookmarkListCreate: GQLEndpoint<BookmarkListCreateInput, CreateOneResult<BookmarkList>>;
        bookmarkListUpdate: GQLEndpoint<BookmarkListUpdateInput, UpdateOneResult<BookmarkList>>;
    }
} = {
    BookmarkListSortBy,
    Query: {
        bookmarkList: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        bookmarkLists: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        bookmarkListCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        bookmarkListUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
import { BookmarkListSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { BookmarkListEndpoints, EndpointsBookmarkList } from "../logic";

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
        bookmarksContainsId: ID
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
`;

export const resolvers: {
    BookmarkListSortBy: typeof BookmarkListSortBy;
    Query: EndpointsBookmarkList["Query"];
    Mutation: EndpointsBookmarkList["Mutation"];
} = {
    BookmarkListSortBy,
    ...BookmarkListEndpoints,
};

import { BookmarkFor, BookmarkSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { BookmarkEndpoints, EndpointsBookmark } from "../logic/bookmark";
import { resolveUnion } from "./resolvers";

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
        forConnect: ID!
        listConnect: ID
        listCreate: BookmarkListCreateInput
    }
    input BookmarkUpdateInput {
        id: ID!
        listConnect: ID
        listUpdate: BookmarkListUpdateInput
    }
    type Bookmark {
        id: ID!
        created_at: Date!
        updated_at: Date!
        by: User!
        list: BookmarkList!
        to: BookmarkTo!
    }

    input BookmarkSearchInput {
        after: String
        apiId: ID
        commentId: ID
        excludeLinkedToTag: Boolean
        ids: [ID!]
        issueId: ID
        limitTo: [BookmarkFor!]
        listLabel: String
        listId: ID
        noteId: ID
        organizationId: ID
        postId: ID
        projectId: ID
        questionId: ID
        questionAnswerId: ID
        quizId: ID
        routineId: ID
        searchString: String
        smartContractId: ID
        sortBy: BookmarkSortBy
        standardId: ID
        tagId: ID
        take: Int
        userId: ID
        visibility: VisibilityType
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
`;

export const resolvers: {
    BookmarkSortBy: typeof BookmarkSortBy;
    BookmarkFor: typeof BookmarkFor;
    BookmarkTo: UnionResolver;
    Query: EndpointsBookmark["Query"];
    Mutation: EndpointsBookmark["Mutation"];
} = {
    BookmarkSortBy,
    BookmarkFor,
    BookmarkTo: { __resolveType(obj) { return resolveUnion(obj); } },
    ...BookmarkEndpoints,
};

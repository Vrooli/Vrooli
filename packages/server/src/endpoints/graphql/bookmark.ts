import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, BookmarkSortBy, BookmarkUpdateInput, FindByIdInput, VisibilityType } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from "../../types";
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

const objectType = "Bookmark";
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
    BookmarkTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    Query: {
        bookmark: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        bookmarks: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        bookmarkCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        bookmarkUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

import { ReactionFor, ReactionSortBy } from "@local/shared";
import { UnionResolver } from "../../types";
import { EndpointsReaction, ReactionEndpoints } from "../logic/reaction";
import { resolveUnion } from "./resolvers";

export const typeDef = `#graphql
    enum ReactionSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum ReactionFor {
        Api
        ChatMessage
        Code
        Comment
        Issue
        Note
        Post
        Project
        Question
        QuestionAnswer
        Quiz
        Routine
        Standard
    }   

    union ReactionTo = Api | Code | ChatMessage | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | Standard

    input ReactInput {
        emoji: String # Null to delete
        reactionFor: ReactionFor!
        forConnect: ID!
    }
    type Reaction {
        id: ID!
        emoji: String!
        by: User!
        to: ReactionTo!
    }

    type ReactionSummary {
        emoji: String!
        count: Int!
    }

    input ReactionSearchInput {
        after: String
        apiId: ID
        chatMessageId: ID
        codeId: ID
        commentId: ID
        excludeLinkedToTag: Boolean
        ids: [ID!]
        issueId: ID
        noteId: ID
        postId: ID
        projectId: ID
        questionId: ID
        questionAnswerId: ID
        quizId: ID
        routineId: ID
        searchString: String
        sortBy: ReactionSortBy
        standardId: ID
        take: Int
    }

    type ReactionSearchResult {
        pageInfo: PageInfo!
        edges: [ReactionEdge!]!
    }

    type ReactionEdge {
        cursor: String!
        node: Reaction!
    }

    extend type Query {
        reactions(input: ReactionSearchInput!): ReactionSearchResult!
    }

    extend type Mutation {
        react(input: ReactInput!): Success!
    }
`;

export const resolvers: {
    ReactionSortBy: typeof ReactionSortBy,
    ReactionFor: typeof ReactionFor,
    ReactionTo: UnionResolver,
    Query: EndpointsReaction["Query"],
    Mutation: EndpointsReaction["Mutation"],
} = {
    ReactionSortBy,
    ReactionFor,
    ReactionTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...ReactionEndpoints,
};

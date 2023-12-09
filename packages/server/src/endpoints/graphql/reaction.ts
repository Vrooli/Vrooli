import { ReactionFor, ReactionSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsReaction, ReactionEndpoints } from "../logic/reaction";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum ReactionSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum ReactionFor {
        Api
        ChatMessage
        Comment
        Issue
        Note
        Post
        Project
        Question
        QuestionAnswer
        Quiz
        Routine
        SmartContract
        Standard
    }   

    union ReactionTo = Api | ChatMessage | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard

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
        excludeLinkedToTag: Boolean
        ids: [ID!]
        searchString: String
        sortBy: ReactionSortBy
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

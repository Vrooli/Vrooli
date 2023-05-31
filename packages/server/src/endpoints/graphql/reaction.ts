import { ReactInput, Reaction, ReactionFor, ReactionSearchInput, ReactionSortBy, Success } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { rateLimit } from "../../middleware";
import { ReactionModel } from "../../models";
import { FindManyResult, GQLEndpoint, UnionResolver } from "../../types";
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

const objectType = "Reaction";
export const resolvers: {
    ReactionSortBy: typeof ReactionSortBy,
    ReactionFor: typeof ReactionFor,
    ReactionTo: UnionResolver,
    Query: {
        reactions: GQLEndpoint<ReactionSearchInput, FindManyResult<Reaction>>;
    },
    Mutation: {
        react: GQLEndpoint<ReactInput, Success>;
    }
} = {
    ReactionSortBy,
    ReactionFor,
    ReactionTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    Query: {
        reactions: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a reaction on an object. A user can only have one reaction per object, meaning 
         * the previous reaction is overruled
         */
        react: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const success = await ReactionModel.react(prisma, userData, input);
            return { __typename: "Success", success };
        },
    },
};

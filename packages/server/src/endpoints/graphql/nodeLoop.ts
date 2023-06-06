import { gql } from "apollo-server-express";

export const typeDef = gql`
    input NodeLoopCreateInput {
        id: ID!
        loops: Int
        maxLoops: Int
        operation: String
        nodeConnect: ID!
        whilesCreate: [NodeLoopWhileCreateInput!]
    }
    input NodeLoopUpdateInput {
        id: ID!
        loops: Int
        maxLoops: Int
        operation: String
        nodeConnect: ID
        whilesCreate: [NodeLoopWhileCreateInput!]
        whilesUpdate: [NodeLoopWhileUpdateInput!]
        whilesDelete: [ID!]
    }
    type NodeLoop {
        id: ID!
        loops: Int
        operation: String
        maxLoops: Int
        whiles: [NodeLoopWhile!]!
    }
`;

export const resolvers = {};

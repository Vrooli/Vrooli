import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';
import { NodeModel } from '../models';
import pkg from '@prisma/client';
const { NodeType } = pkg;

export const typeDef = gql`
    enum NodeType {
        COMBINE
        DECISION
        END
        LOOP
        ROUTINE_LIST
        REDIRECT
        START
    }

    union NodeData = NodeCombine | NodeDecision | NodeEnd | NodeLoop | NodeRoutineList | NodeRedirect | NodeStart

    input NodeInput {
        id: ID
        routineId: ID
        title: String
        description: String
        type: NodeType
        combineData: NodeCombineInput
        decisionData: NodeDecisionInput
        endData: NodeEndInput
        loopData: NodeLoopInput
        routineListData: NodeRoutineListInput
        redirectData: NodeRedirectInput
        startData: NodeStartInput
    }

    type Node {
        id: ID!
        routineId: ID!
        title: String!
        description: String
        type: NodeType!
        data: NodeData
        routine: Routine!
    }

    input NodeCombineInput {
        id: ID
        from: [NodeCombineFromInput!]!
        to: NodeInput
    }

    type NodeCombine {
        id: ID!
        from: [NodeCombineFrom!]!
        to: Node
    }

    input NodeCombineFromInput {
        id: ID
        combineId: ID
        fromId: ID
    }

    type NodeCombineFrom {
        id: ID!
        combineId: ID!
        fromId: ID!
        combine: NodeCombine
        from: Node
    }

    input NodeDecisionInput {
        id: ID
        decisions: [NodeDecisionItemInput!]!
    }

    type NodeDecision {
        id: ID!
        decisions: [NodeDecisionItem!]!
    }

    input NodeDecisionItemInput {
        id: ID
        title: String
        when: [NodeDecisionItemCaseInput]
    }

    type NodeDecisionItem {
        id: ID!
        title: String!
        when: [NodeDecisionItemCase]!
    }

    input NodeDecisionItemCaseInput {
        id: ID
        condition: String
    }

    type NodeDecisionItemCase {
        id: ID!
        condition: String!
    }

    input NodeEndInput {
        id: ID
    }

    type NodeEnd {
        id: ID!
    }

    input NodeLoopInput {
        id: ID
    }

    type NodeLoop {
        id: ID!
    }

    input NodeRoutineListInput {
        id: ID
        isOrdered: Boolean
        routines: [NodeRoutineListItemInput!]!
    }

    type NodeRoutineList {
        id: ID!
        isOrdered: Boolean!
        routines: [NodeRoutineListItem!]!
    }

    input NodeRoutineListItemInput {
        id: ID
        title: String
        description: String
        isOptional: Boolean
        listId: ID
        routineId: ID
    }

    type NodeRoutineListItem {
        id: ID!
        title: String!
        description: String
        isOptional: Boolean!
        list: NodeRoutineList
        routine: Routine
    }

    input NodeRedirectInput {
        id: ID
    }

    type NodeRedirect {
        id: ID!
    }

    input NodeStartInput {
        id: ID
    }

    type NodeStart {
        id: ID!
    }

    extend type Mutation {
        addNode(input: NodeInput!): Node!
        updateNode(input: NodeInput!): Node!
        deleteNode(input: DeleteOneInput!): Boolean!
    }
`

export const resolvers = {
    NodeType: NodeType,
    Mutation: {
        /**
         * Updates an individual node in a routine orchestration.
         * Note that the order of a routine (i.e. previous, next) cannot be updated with this mutation. 
         * @returns Updated node
         */
        addNode: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            // Add node
            return await new NodeModel(context.prisma).create(input, info);
        },
        updateNode: async (_parent: undefined, { input }: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        deleteNode: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        }
    }
}
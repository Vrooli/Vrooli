import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { NodeModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Node, NodeInput } from './types';
import { Context } from '../context';
import pkg from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
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
        previous: Node
        next: Node
        To: [Node!]!
        From: [Node!]!
        Previous: [Node!]!
        Next: [Node!]!
        DecisionItem: [NodeDecisionItem!]!
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
        addNode: async (_parent: undefined, { input }: IWrap<NodeInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Node> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await NodeModel(prisma).create(input, info);
        },
        updateNode: async (_parent: undefined, { input }: IWrap<NodeInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await NodeModel(prisma).update(input, info);
        },
        deleteNode: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await NodeModel(prisma).delete(input);
        }
    }
}
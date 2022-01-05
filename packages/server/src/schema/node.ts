import { gql } from 'apollo-server-express';
import { CODE, NodeType } from '@local/shared';
import { CustomError } from '../error';
import { NodeModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Node, NodeInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum NodeType {
        Combine
        Decision
        End
        Loop
        RoutineList
        Redirect
        Start
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
        created_at: Date!
        updated_at: Date!
        routineId: ID!
        title: String!
        description: String
        type: NodeType!
        data: NodeData
        routine: Routine!
        previous: ID
        next: ID
        To: [Node!]!
        From: [Node!]!
        Previous: [Node!]!
        Next: [Node!]!
        DecisionItem: [NodeDecisionItem!]!
    }

    input NodeCombineInput {
        id: ID
        from: [ID!]
        to: ID
    }

    type NodeCombine {
        id: ID!
        from: [ID!]!
        to: ID!
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
        description: String
        toId: ID
        when: [NodeDecisionItemCaseInput]
    }

    type NodeDecisionItem {
        id: ID!
        title: String!
        description: String
        toId: ID
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
        wasSuccessful: Boolean
    }

    type NodeEnd {
        id: ID!
        wasSuccessful: Boolean!
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
        isOptional: Boolean
        routines: [NodeRoutineListItemInput!]!
    }

    type NodeRoutineList {
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
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
        nodeAdd(input: NodeInput!): Node!
        nodeUpdate(input: NodeInput!): Node!
        nodeDeleteOne(input: DeleteOneInput!): Success!
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
        nodeAdd: async (_parent: undefined, { input }: IWrap<NodeInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await NodeModel(prisma).create(input, info);
            // Format object to GraphQL type
            return NodeModel().toGraphQL(dbModel);
        },
        nodeUpdate: async (_parent: undefined, { input }: IWrap<NodeInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // Update object
            const dbModel = await NodeModel(prisma).update(input, info);
            // Format to GraphQL type
            return NodeModel().toGraphQL(dbModel);
        },
        nodeDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await NodeModel(prisma).delete(input);
            return { success };
        }
    }
}
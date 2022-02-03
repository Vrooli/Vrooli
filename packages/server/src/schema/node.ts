import { gql } from 'apollo-server-express';
import { CODE, NodeType } from '@local/shared';
import { CustomError } from '../error';
import { nodeFormatter, NodeModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Node, NodeAddInput, NodeUpdateInput, Success } from './types';
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

    input NodeAddInput {
        description: String
        title: String
        type: NodeType
        nodeCombineAdd: NodeCombineAddInput
        nodeDecisionAdd: NodeDecisionAddInput
        nodeEndAdd: NodeEndAddInput
        nodeLoopAdd: NodeLoopAddInput
        nodeRedirectAdd: NodeRoutineListAddInput
        nodeRoutineListAdd: NodeRedirectAddInput
        nodeStartAdd: NodeStartAddInput
        previousId: ID
        nextId: ID
        routineId: ID!
    }
    input NodeUpdateInput {
        id: ID!
        description: String
        title: String
        type: NodeType
        nodeCombineAdd: NodeCombineAddInput
        nodeCombineUpdate: NodeCombineUpdateInput
        nodeDecisionAdd: NodeDecisionAddInput
        nodeDecisionUpdate: NodeDecisionUpdateInput
        nodeEndAdd: NodeEndAddInput
        nodeEndUpdate: NodeEndUpdateInput
        nodeLoopAdd: NodeLoopAddInput
        nodeLoopUpdate: NodeLoopUpdateInput
        nodeRedirectAdd: NodeRoutineListAddInput
        nodeRedirectUpdate: NodeRoutineListUpdateInput
        nodeRoutineListAdd: NodeRedirectAddInput
        nodeRoutineListUpdate: NodeRedirectUpdateInput
        nodeStartAdd: NodeStartAddInput
        nodeStartUpdate: NodeStartUpdateInput
        previousId: ID
        nextId: ID
        routineId: ID
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

    input NodeCombineAddInput {
        from: [ID!]!
        toId: ID
    }
    input NodeCombineUpdateInput {
        id: ID!
        from: [ID!]
        toId: ID
    }
    type NodeCombine {
        id: ID!
        from: [ID!]!
        toId: ID!
    }

    input NodeDecisionAddInput {
        decisionsAdd: [NodeDecisionItemAddInput!]!
    }
    input NodeDecisionUpdateInput {
        id: ID!
        decisionsAdd: [NodeDecisionItemAddInput!]
        decisionsUpdate: [NodeDecisionItemUpdateInput!]
        decisionsDelete: [ID!]
    }
    type NodeDecision {
        id: ID!
        decisions: [NodeDecisionItem!]!
    }
    input NodeDecisionItemAddInput {
        description: String
        title: String!
        whenAdd: [NodeDecisionItemWhenAddInput!]!
        toId: ID
    }
    input NodeDecisionItemUpdateInput {
        id: ID!
        description: String
        title: String
        whenAdd: [NodeDecisionItemWhenAddInput!]
        whenUpdate: [NodeDecisionItemWhenUpdateInput!]
        whenDelete: [ID!]
        toId: ID
    }
    type NodeDecisionItem {
        id: ID!
        description: String
        title: String!
        when: [NodeDecisionItemWhen!]!
        toId: ID
    } 
    input NodeDecisionItemWhenAddInput {
        condition: String!
    }
    input NodeDecisionItemWhenUpdateInput {
        id: ID!
        condition: String
    }
    type NodeDecisionItemWhen {
        id: ID!
        condition: String!
    }

    input NodeEndAddInput {
        wasSuccessful: Boolean
    }
    input NodeEndUpdateInput {
        id: ID!
        wasSuccessful: Boolean
    }
    type NodeEnd {
        id: ID!
        wasSuccessful: Boolean!
    }

    input NodeLoopAddInput {
        id: ID
    }
    input NodeLoopUpdateInput {
        id: ID
    }
    type NodeLoop {
        id: ID!
    }

    input NodeRedirectAddInput {
        id: ID
    }
    input NodeRedirectUpdateInput {
        id: ID
    }
    type NodeRedirect {
        id: ID!
    }

    input NodeRoutineListAddInput {
        isOrdered: Boolean
        isOptional: Boolean
        routinesConnect: [ID!]
        routinesAdd: [NodeRoutineListItemAddInput!]
    }
    input NodeRoutineListUpdateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        routinesConnect: [ID!]
        routinesDisconnect: [ID!]
        routinesDelete: [ID!]
        routinesAdd: [NodeRoutineListItemAddInput!]
        routinesUpdate: [NodeRoutineListItemUpdateInput!]
    }
    type NodeRoutineList {
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
        routines: [NodeRoutineListItem!]!
    }
    input NodeRoutineListItemAddInput {
        description: String
        isOptional: Boolean
        title: String
        routineConnect: ID!
    }
    input NodeRoutineListItemUpdateInput {
        id: ID!
        description: String
        isOptional: Boolean
        title: String
        routineConnect: ID
    }
    type NodeRoutineListItem {
        id: ID!
        description: String
        isOptional: Boolean!
        title: String
        routine: Routine!
    }

    input NodeStartAddInput {
        _blank: String
    }
    input NodeStartUpdateInput {
        id: ID!
    }
    type NodeStart {
        id: ID!
    }

    extend type Mutation {
        nodeAdd(input: NodeAddInput!): Node!
        nodeUpdate(input: NodeUpdateInput!): Node!
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
        nodeAdd: async (_parent: undefined, { input }: IWrap<NodeAddInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await NodeModel(prisma).addNode(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        nodeUpdate: async (_parent: undefined, { input }: IWrap<NodeUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await NodeModel(prisma).updateNode(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        nodeDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await NodeModel(prisma).deleteNode(req.userId, input);
        }
    }
}
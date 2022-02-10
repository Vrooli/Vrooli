import { gql } from 'apollo-server-express';
import { CODE, NodeType } from '@local/shared';
import { CustomError } from '../error';
import { NodeModel } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Node, NodeCreateInput, NodeUpdateInput, Success } from './types';
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

    union NodeData = NodeCombine | NodeDecision | NodeEnd | NodeLoop | NodeRoutineList

    input NodeCreateInput {
        description: String
        title: String
        type: NodeType
        nodeCombineCreate: NodeCombineCreateInput
        nodeDecisionCreate: NodeDecisionCreateInput
        nodeEndCreate: NodeEndCreateInput
        nodeLoopCreate: NodeLoopCreateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        previousId: ID
        nextId: ID
        routineId: ID!
    }
    input NodeUpdateInput {
        id: ID!
        description: String
        title: String
        type: NodeType
        nodeCombineCreate: NodeCombineCreateInput
        nodeCombineUpdate: NodeCombineUpdateInput
        nodeDecisionCreate: NodeDecisionCreateInput
        nodeDecisionUpdate: NodeDecisionUpdateInput
        nodeEndCreate: NodeEndCreateInput
        nodeEndUpdate: NodeEndUpdateInput
        nodeLoopCreate: NodeLoopCreateInput
        nodeLoopUpdate: NodeLoopUpdateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        nodeRoutineListUpdate: NodeRoutineListUpdateInput
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

    input NodeCombineCreateInput {
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

    input NodeDecisionCreateInput {
        decisionsCreate: [NodeDecisionItemCreateInput!]!
    }
    input NodeDecisionUpdateInput {
        id: ID!
        decisionsCreate: [NodeDecisionItemCreateInput!]
        decisionsUpdate: [NodeDecisionItemUpdateInput!]
        decisionsDelete: [ID!]
    }
    type NodeDecision {
        id: ID!
        decisions: [NodeDecisionItem!]!
    }
    input NodeDecisionItemCreateInput {
        description: String
        title: String!
        whenCreate: [NodeDecisionItemWhenCreateInput!]!
        toId: ID
    }
    input NodeDecisionItemUpdateInput {
        id: ID!
        description: String
        title: String
        whenCreate: [NodeDecisionItemWhenCreateInput!]
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
    input NodeDecisionItemWhenCreateInput {
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

    input NodeEndCreateInput {
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

    input NodeLoopCreateInput {
        loops: Int
        maxLoops: Int
        whilesCreate: [NodeLoopWhileCreateInput!]!
    }
    input NodeLoopUpdateInput {
        id: ID!
        loops: Int
        maxLoops: Int
        whilesCreate: [NodeLoopWhileCreateInput!]!
        whilesUpdate: [NodeLoopWhileUpdateInput!]!
        whilesDelete: [ID!]
    }
    type NodeLoop {
        id: ID!
        loops: Int
        maxLoops: Int
        whiles: [NodeLoopWhile!]!
    }
    input NodeLoopWhileCreateInput {
        description: String
        title: String!
        whenCreate: [NodeLoopWhileCaseCreateInput!]!
        toId: ID
    }
    input NodeLoopWhileUpdateInput {
        id: ID!
        description: String
        title: String
        whenCreate: [NodeLoopWhileCaseCreateInput!]
        whenUpdate: [NodeLoopWhileCaseUpdateInput!]
        whenDelete: [ID!]
        toId: ID
    }
    type NodeLoopWhile {
        id: ID!
        description: String
        title: String!
        when: [NodeLoopWhileCase!]!
        toId: ID
    } 
    input NodeLoopWhileCaseCreateInput {
        condition: String!
    }
    input NodeLoopWhileCaseUpdateInput {
        id: ID!
        condition: String
    }
    type NodeLoopWhileCase {
        id: ID!
        condition: String!
    }

    input NodeRoutineListCreateInput {
        isOrdered: Boolean
        isOptional: Boolean
        routinesConnect: [ID!]
        routinesCreate: [NodeRoutineListItemCreateInput!]
    }
    input NodeRoutineListUpdateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        routinesConnect: [ID!]
        routinesDisconnect: [ID!]
        routinesDelete: [ID!]
        routinesCreate: [NodeRoutineListItemCreateInput!]
        routinesUpdate: [NodeRoutineListItemUpdateInput!]
    }
    type NodeRoutineList {
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
        routines: [NodeRoutineListItem!]!
    }
    input NodeRoutineListItemCreateInput {
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

    extend type Mutation {
        nodeCreate(input: NodeCreateInput!): Node!
        nodeUpdate(input: NodeUpdateInput!): Node!
        nodeDeleteOne(input: DeleteOneInput!): Success!
    }
`

export const resolvers = {
    NodeType: NodeType,
    NodeData: {
        __resolveType(obj: any) {
            console.log('IN NODEDATA __resolveType', obj);
            // Only NodeCombine has from and toId fields
            if (obj.hasOwnProperty('from') && obj.hasOwnProperty('toId')) return 'NodeCombine';
            // Only NodeDecision has decisions field
            if (obj.hasOwnProperty('decisions')) return 'NodeDecision';
            // Only NodeEnd has wasSuccessful field
            if (obj.hasOwnProperty('wasSuccessful')) return 'NodeEnd';
            // Only NodeLoop has loop field
            if (obj.hasOwnProperty('loop')) return 'NodeLoop';
            // Only NodeRoutineList has isOrdered field
            if (obj.hasOwnProperty('isOrdered')) return 'NodeRoutineList';
            return null; // GraphQLError is thrown
        },
    },
    Mutation: {
        /**
         * Updates an individual node in a routine orchestration.
         * Note that the order of a routine (i.e. previous, next) cannot be updated with this mutation. 
         * @returns Updated node
         */
        nodeCreate: async (_parent: undefined, { input }: IWrap<NodeCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const created = await NodeModel(prisma).create(req.userId, input, info);
            if (!created) throw new CustomError(CODE.ErrorUnknown);
            return created;
        },
        nodeUpdate: async (_parent: undefined, { input }: IWrap<NodeUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const updated = await NodeModel(prisma).update(req.userId, input, info);
            if (!updated) throw new CustomError(CODE.ErrorUnknown);
            return updated;
        },
        nodeDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.userId) throw new CustomError(CODE.Unauthorized);
            return await NodeModel(prisma).delete(req.userId, input);
        }
    }
}
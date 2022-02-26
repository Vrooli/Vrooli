import { gql } from 'apollo-server-express';
import { createHelper, deleteOneHelper, NodeModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Node, NodeCreateInput, NodeUpdateInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import pkg from '@prisma/client';
const { NodeType } = pkg;

export const typeDef = gql`
    enum NodeType {
        End
        Loop
        RoutineList
        Redirect
        Start
    }

    union NodeData = NodeEnd | NodeLoop | NodeRoutineList

    input NodeCreateInput {
        description: String
        title: String
        type: NodeType
        nodeEndCreate: NodeEndCreateInput
        nodeLoopCreate: NodeLoopCreateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        routineId: ID!
    }
    input NodeUpdateInput {
        id: ID!
        description: String
        title: String
        type: NodeType
        nodeEndCreate: NodeEndCreateInput
        nodeEndUpdate: NodeEndUpdateInput
        nodeLoopCreate: NodeLoopCreateInput
        nodeLoopUpdate: NodeLoopUpdateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        nodeRoutineListUpdate: NodeRoutineListUpdateInput
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

    input NodeLinkCreateInput {
        conditions: [NodeLinkConditionCreateInput!]!
        fromId: ID!
        toId: ID!
    }
    input NodeLinkUpdateInput {
        id: ID!
        conditionsCreate: [NodeLinkConditionCreateInput!]
        conditionsUpdate: [NodeLinkConditionUpdateInput!]
        conditionsDelete: [ID!]
        fromId: ID
        toId: ID
    }
    type NodeLink{
        id: ID!
        conditions: [NodeLinkCondition!]!
        fromId: ID!
        toId: ID!
    }
    input NodeLinkConditionCreateInput {
        description: String
        title: String!
        whenCreate: [NodeLinkConditionCaseCreateInput!]!
        toId: ID
    }
    input NodeLinkConditionUpdateInput {
        id: ID!
        description: String
        title: String
        whenCreate: [NodeLinkConditionCaseCreateInput!]
        whenUpdate: [NodeLinkConditionCaseUpdateInput!]
        whenDelete: [ID!]
        toId: ID
    }
    type NodeLinkCondition {
        id: ID!
        description: String
        title: String!
        when: [NodeLinkConditionCase!]!
    } 
    input NodeLinkConditionCaseCreateInput {
        condition: String!
    }
    input NodeLinkConditionCaseUpdateInput {
        id: ID!
        condition: String
    }
    type NodeLinkConditionCase {
        id: ID!
        condition: String!
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
            return createHelper(req.userId, input, info, NodeModel(prisma).cud);
        },
        nodeUpdate: async (_parent: undefined, { input }: IWrap<NodeUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            return updateHelper(req.userId, input, info, NodeModel(prisma).cud);
        },
        nodeDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, NodeModel(prisma).cud);
        }
    }
}
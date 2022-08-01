import { gql } from 'apollo-server-express';
import { createHelper, NodeModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from 'types';
import { Node, NodeCreateInput, NodeUpdateInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import pkg from '@prisma/client';
import { rateLimit } from '../rateLimit';
import { resolveNodeData } from './resolvers';
const { NodeType } = pkg;

export const typeDef = gql`
    enum NodeType {
        End
        Redirect
        RoutineList
        Start
    }

    union NodeData = NodeEnd | NodeRoutineList

    input NodeCreateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        type: NodeType
        loopCreate: LoopCreateInput
        nodeEndCreate: NodeEndCreateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        routineId: ID
        translationsCreate: [NodeTranslationCreateInput!]
    }
    input NodeUpdateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        type: NodeType
        loopDelete: ID
        loopCreate: LoopCreateInput
        loopUpdate: LoopUpdateInput
        nodeEndUpdate: NodeEndUpdateInput
        nodeRoutineListUpdate: NodeRoutineListUpdateInput
        routineId: ID
        translationsDelete: [ID!]
        translationsCreate: [NodeTranslationCreateInput!]
        translationsUpdate: [NodeTranslationUpdateInput!]
    }
    type Node {
        id: ID!
        created_at: Date!
        updated_at: Date!
        columnIndex: Int
        routineId: ID!
        rowIndex: Int
        type: NodeType!
        loop: Loop
        data: NodeData
        routine: Routine!
        translations: [NodeTranslation!]!
    }

    input NodeTranslationCreateInput {
        id: ID!
        language: String!
        title: String!
        description: String
    }
    input NodeTranslationUpdateInput {
        id: ID!
        language: String
        title: String
        description: String
    }
    type NodeTranslation {
        id: ID!
        language: String!
        title: String!
        description: String
    }

    input NodeEndCreateInput {
        id: ID!
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

    input LoopCreateInput {
        id: ID!
        loops: Int
        maxLoops: Int
        operation: String
        whilesCreate: [LoopWhileCreateInput!]!
    }
    input LoopUpdateInput {
        id: ID!
        loops: Int
        maxLoops: Int
        operation: String
        whilesCreate: [LoopWhileCreateInput!]!
        whilesUpdate: [LoopWhileUpdateInput!]!
        whilesDelete: [ID!]
    }
    type Loop {
        id: ID!
        loops: Int
        operation: String
        maxLoops: Int
        whiles: [LoopWhile!]!
    }

    input LoopWhileCreateInput {
        id: ID!
        translationsCreate: [LoopWhileTranslationCreateInput!]
        condition: String!
        toId: ID
    }
    input LoopWhileUpdateInput {
        id: ID!
        toId: ID
        translationsDelete: [ID!]
        translationsCreate: [LoopWhileTranslationCreateInput!]
        translationsUpdate: [LoopWhileTranslationUpdateInput!]
        condition: String
    }
    type LoopWhile {
        id: ID!
        toId: ID
        translations: [LoopWhileTranslation!]!
        condition: String!
    } 

    input LoopWhileTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String!
    }
    input LoopWhileTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type LoopWhileTranslation {
        id: ID!
        language: String!
        description: String
        title: String!
    }

    input NodeRoutineListCreateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        routinesCreate: [NodeRoutineListItemCreateInput!]
    }
    input NodeRoutineListUpdateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
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
        id: ID!
        index: Int!
        isOptional: Boolean
        routineConnect: ID!
        translationsCreate: [NodeRoutineListItemTranslationCreateInput!]
    }
    input NodeRoutineListItemUpdateInput {
        id: ID!
        index: Int
        isOptional: Boolean
        routineUpdate: RoutineUpdateInput
        translationsDelete: [ID!]
        translationsCreate: [NodeRoutineListItemTranslationCreateInput!]
        translationsUpdate: [NodeRoutineListItemTranslationUpdateInput!]
    }
    type NodeRoutineListItem {
        id: ID!
        index: Int!
        isOptional: Boolean!
        routine: Routine!
        translations: [NodeRoutineListItemTranslation!]!
    }

    input NodeRoutineListItemTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String
    }
    input NodeRoutineListItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type NodeRoutineListItemTranslation {
        id: ID!
        language: String!
        description: String
        title: String
    }

    input NodeLinkCreateInput {
        id: ID!
        whens: [NodeLinkWhenCreateInput!]
        operation: String
        fromId: ID!
        toId: ID!
    }
    input NodeLinkUpdateInput {
        id: ID!
        whensCreate: [NodeLinkWhenCreateInput!]
        whensUpdate: [NodeLinkWhenUpdateInput!]
        whensDelete: [ID!]
        operation: String
        fromId: ID
        toId: ID
    }
    type NodeLink{
        id: ID!
        whens: [NodeLinkWhen!]!
        operation: String
        fromId: ID!
        toId: ID!
    }

    input NodeLinkWhenCreateInput {
        id: ID!
        linkId: ID
        translationsCreate: [NodeLinkWhenTranslationCreateInput!]
        condition: String!
    }
    input NodeLinkWhenUpdateInput {
        id: ID!
        linkId: ID
        translationsDelete: [ID!]
        translationsCreate: [NodeLinkWhenTranslationCreateInput!]
        translationsUpdate: [NodeLinkWhenTranslationUpdateInput!]
        condition: String
    }
    type NodeLinkWhen {
        id: ID!
        translations: [NodeLinkWhenTranslation!]!
        condition: String!
    } 

    input NodeLinkWhenTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String!
    }
    input NodeLinkWhenTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type NodeLinkWhenTranslation {
        id: ID!
        language: String!
        description: String
        title: String!
    }

    extend type Mutation {
        nodeCreate(input: NodeCreateInput!): Node!
        nodeUpdate(input: NodeUpdateInput!): Node!
    }
`

export const resolvers = {
    NodeType: NodeType,
    NodeData: {
        __resolveType(obj: any) { return resolveNodeData(obj) }
    },
    Mutation: {
        /**
         * Updates an individual node in a routine orchestration.
         * Note that the order of a routine (i.e. previous, next) cannot be updated with this mutation. 
         * @returns Updated node
         */
        nodeCreate: async (_parent: undefined, { input }: IWrap<NodeCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            await rateLimit({ info, max: 2000, byAccountOrKey: true, req });
            return createHelper({ info, input, model: NodeModel, prisma, userId: req.userId })
        },
        nodeUpdate: async (_parent: undefined, { input }: IWrap<NodeUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Node>> => {
            await rateLimit({ info, max: 2000, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: NodeModel, prisma, userId: req.userId })
        },
    }
}
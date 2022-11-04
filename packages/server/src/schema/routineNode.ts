import { gql } from 'apollo-server-express';
import { createHelper, RoutineNodeModel, updateHelper } from '../models';
import { IWrap, RecursivePartial } from '../types';
import { RoutineNode, RoutineNodeCreateInput, RoutineNodeUpdateInput } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import pkg from '@prisma/client';
import { rateLimit } from '../rateLimit';
import { resolveRoutineNodeData } from './resolvers';
const { NodeType } = pkg;

export const typeDef = gql`
    enum RoutineNodeType {
        End
        Redirect
        RoutineList
        Start
    }

    union RoutineNodeData = RoutineNodeEnd | RoutineNodeRoutineList

    input RoutineNodeCreateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        type: RoutineNodeType
        loopCreate: LoopCreateInput
        nodeEndCreate: RoutineNodeEndCreateInput
        nodeRoutineListCreate: RoutineNodeRoutineListCreateInput
        routineId: ID
        translationsCreate: [RoutineNodeTranslationCreateInput!]
    }
    input RoutineNodeUpdateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        type: RoutineNodeType
        loopDelete: ID
        loopCreate: LoopCreateInput
        loopUpdate: LoopUpdateInput
        nodeEndCreate: RoutineNodeEndCreateInput
        nodeEndUpdate: RoutineNodeEndUpdateInput
        nodeRoutineListCreate: RoutineNodeRoutineListCreateInput
        nodeRoutineListUpdate: RoutineNodeRoutineListUpdateInput
        routineId: ID
        translationsDelete: [ID!]
        translationsCreate: [RoutineNodeTranslationCreateInput!]
        translationsUpdate: [RoutineNodeTranslationUpdateInput!]
    }
    type RoutineNode {
        id: ID!
        created_at: DateTime!
        updated_at: DateTime!
        columnIndex: Int
        routineId: ID!
        rowIndex: Int
        type: RoutineNodeType!
        loop: Loop
        data: RoutineNodeData
        routine: Routine!
        translations: [RoutineNodeTranslation!]!
    }

    input RoutineNodeTranslationCreateInput {
        id: ID!
        language: String!
        title: String!
        description: String
    }
    input RoutineNodeTranslationUpdateInput {
        id: ID!
        language: String
        title: String
        description: String
    }
    type RoutineNodeTranslation {
        id: ID!
        language: String!
        title: String!
        description: String
    }

    input RoutineNodeEndCreateInput {
        id: ID!
        wasSuccessful: Boolean
    }
    input RoutineNodeEndUpdateInput {
        id: ID!
        wasSuccessful: Boolean
    }
    type RoutineNodeEnd {
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

    input RoutineNodeRoutineListCreateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        routinesCreate: [RoutineNodeRoutineListItemCreateInput!]
    }
    input RoutineNodeRoutineListUpdateInput {
        id: ID!
        isOrdered: Boolean
        isOptional: Boolean
        routinesDelete: [ID!]
        routinesCreate: [RoutineNodeRoutineListItemCreateInput!]
        routinesUpdate: [RoutineNodeRoutineListItemUpdateInput!]
    }
    type RoutineNodeRoutineList {
        id: ID!
        isOrdered: Boolean!
        isOptional: Boolean!
        routines: [RoutineNodeRoutineListItem!]!
    }

    input RoutineNodeRoutineListItemCreateInput {
        id: ID!
        index: Int!
        isOptional: Boolean
        routineConnect: ID!
        translationsCreate: [RoutineNodeRoutineListItemTranslationCreateInput!]
    }
    input RoutineNodeRoutineListItemUpdateInput {
        id: ID!
        index: Int
        isOptional: Boolean
        routineUpdate: RoutineUpdateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineNodeRoutineListItemTranslationCreateInput!]
        translationsUpdate: [RoutineNodeRoutineListItemTranslationUpdateInput!]
    }
    type RoutineNodeRoutineListItem {
        id: ID!
        index: Int!
        isOptional: Boolean!
        routine: Routine!
        translations: [RoutineNodeRoutineListItemTranslation!]!
    }

    input RoutineNodeRoutineListItemTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String
    }
    input RoutineNodeRoutineListItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type RoutineNodeRoutineListItemTranslation {
        id: ID!
        language: String!
        description: String
        title: String
    }

    input RoutineNodeLinkCreateInput {
        id: ID!
        whens: [RoutineNodeLinkWhenCreateInput!]
        operation: String
        fromId: ID!
        toId: ID!
    }
    input RoutineNodeLinkUpdateInput {
        id: ID!
        whensCreate: [RoutineNodeLinkWhenCreateInput!]
        whensUpdate: [RoutineNodeLinkWhenUpdateInput!]
        whensDelete: [ID!]
        operation: String
        fromId: ID
        toId: ID
    }
    type RoutineNodeLink{
        id: ID!
        whens: [RoutineNodeLinkWhen!]!
        operation: String
        fromId: ID!
        toId: ID!
    }

    input RoutineNodeLinkWhenCreateInput {
        id: ID!
        linkId: ID
        translationsCreate: [RoutineNodeLinkWhenTranslationCreateInput!]
        condition: String!
    }
    input RoutineNodeLinkWhenUpdateInput {
        id: ID!
        linkId: ID
        translationsDelete: [ID!]
        translationsCreate: [RoutineNodeLinkWhenTranslationCreateInput!]
        translationsUpdate: [RoutineNodeLinkWhenTranslationUpdateInput!]
        condition: String
    }
    type RoutineNodeLinkWhen {
        id: ID!
        translations: [RoutineNodeLinkWhenTranslation!]!
        condition: String!
    } 

    input RoutineNodeLinkWhenTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        title: String!
    }
    input RoutineNodeLinkWhenTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        title: String
    }
    type RoutineNodeLinkWhenTranslation {
        id: ID!
        language: String!
        description: String
        title: String!
    }

    extend type Mutation {
        routineNodeCreate(input: RoutineNodeCreateInput!): RoutineNode!
        routineNodeUpdate(input: RoutineNodeUpdateInput!): RoutineNode!
    }
`

export const resolvers = {
    RoutineNodeType: NodeType,
    RoutineNodeData: {
        __resolveType(obj: any) { return resolveRoutineNodeData(obj) }
    },
    Mutation: {
        /**
         * Updates an individual node in a routine orchestration.
         * Note that the order of a routine (i.e. previous, next) cannot be updated with this mutation. 
         * @returns Updated node
         */
        routineNodeCreate: async (_parent: undefined, { input }: IWrap<RoutineNodeCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<RoutineNode>> => {
            await rateLimit({ info, maxUser: 2000, req });
            return createHelper({ info, input, model: RoutineNodeModel, prisma, req })
        },
        routineNodeUpdate: async (_parent: undefined, { input }: IWrap<RoutineNodeUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<RoutineNode>> => {
            await rateLimit({ info, maxUser: 2000, req });
            return updateHelper({ info, input, model: RoutineNodeModel, prisma, req })
        },
    }
}
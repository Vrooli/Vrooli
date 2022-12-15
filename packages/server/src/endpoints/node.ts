import { gql } from 'apollo-server-express';
import { createHelper, updateHelper } from '../actions';
import { CreateOneResult, GQLEndpoint, UnionResolver, UpdateOneResult } from '../types';
import { Node, NodeCreateInput, NodeUpdateInput } from './types';
import { rateLimit } from '../middleware';
import pkg from '@prisma/client';
import { resolveUnion } from './resolvers';
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
        type: NodeType!
        loopCreate: NodeLoopCreateInput
        nodeEndCreate: NodeEndCreateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        routineVersionConnect: ID!
        translationsCreate: [NodeTranslationCreateInput!]
    }
    input NodeUpdateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        type: NodeType
        loopDelete: ID
        loopCreate: NodeLoopCreateInput
        loopUpdate: NodeLoopUpdateInput
        nodeEndCreate: NodeEndCreateInput
        nodeEndUpdate: NodeEndUpdateInput
        nodeRoutineListCreate: NodeRoutineListCreateInput
        nodeRoutineListUpdate: NodeRoutineListUpdateInput
        routineVersionConnect: ID
        translationsDelete: [ID!]
        translationsCreate: [NodeTranslationCreateInput!]
        translationsUpdate: [NodeTranslationUpdateInput!]
    }
    type Node {
        id: ID!
        created_at: Date!
        updated_at: Date!
        columnIndex: Int
        routineVersionId: ID!
        rowIndex: Int
        type: NodeType!
        loop: NodeLoop
        data: NodeData
        routineVersion: RoutineVersion!
        translations: [NodeTranslation!]!
    }

    input NodeTranslationCreateInput {
        id: ID!
        language: String!
        name: String!
        description: String
    }
    input NodeTranslationUpdateInput {
        id: ID!
        language: String
        name: String
        description: String
    }
    type NodeTranslation {
        id: ID!
        language: String!
        name: String!
        description: String
    }

    extend type Mutation {
        nodeCreate(input: NodeCreateInput!): Node!
        nodeUpdate(input: NodeUpdateInput!): Node!
    }
`
const objectType = 'Node';
export const resolvers: {
    NodeType: typeof NodeType;
    NodeData: UnionResolver,
    Mutation: {
        nodeCreate: GQLEndpoint<NodeCreateInput, CreateOneResult<Node>>;
        nodeUpdate: GQLEndpoint<NodeUpdateInput, UpdateOneResult<Node>>;
    }
} = {
    NodeType: NodeType,
    NodeData: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Mutation: {
        nodeCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        nodeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 2000, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
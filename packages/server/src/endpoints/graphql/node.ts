import pkg from "@prisma/client";
import { EndpointsNode, NodeEndpoints } from "../logic/node";

const { NodeType } = pkg;

export const typeDef = `#graphql
    enum NodeType {
        End
        Redirect
        RoutineList
        Start
    }

    input NodeCreateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        nodeType: NodeType!
        loopCreate: NodeLoopCreateInput
        endCreate: NodeEndCreateInput
        routineListCreate: NodeRoutineListCreateInput
        routineVersionConnect: ID!
        translationsCreate: [NodeTranslationCreateInput!]
    }
    input NodeUpdateInput {
        id: ID!
        columnIndex: Int
        rowIndex: Int
        nodeType: NodeType
        loopDelete: Boolean
        loopCreate: NodeLoopCreateInput
        loopUpdate: NodeLoopUpdateInput
        endCreate: NodeEndCreateInput
        endUpdate: NodeEndUpdateInput
        routineListCreate: NodeRoutineListCreateInput
        routineListUpdate: NodeRoutineListUpdateInput
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
        nodeType: NodeType!
        rowIndex: Int
        end: NodeEnd
        loop: NodeLoop
        routineList: NodeRoutineList
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
        language: String!
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
`;

export const resolvers: {
    NodeType: typeof NodeType;
    Mutation: EndpointsNode["Mutation"];
} = {
    NodeType,
    ...NodeEndpoints,
};

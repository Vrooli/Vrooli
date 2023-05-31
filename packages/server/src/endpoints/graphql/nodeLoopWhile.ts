import { gql } from "apollo-server-express";

export const typeDef = gql`
    input NodeLoopWhileCreateInput {
        id: ID!
        condition: String!
        loopConnect: ID!
        translationsCreate: [NodeLoopWhileTranslationCreateInput!]
    }
    input NodeLoopWhileUpdateInput {
        id: ID!
        condition: String
        translationsDelete: [ID!]
        translationsCreate: [NodeLoopWhileTranslationCreateInput!]
        translationsUpdate: [NodeLoopWhileTranslationUpdateInput!]
    }
    type NodeLoopWhile {
        id: ID!
        condition: String!
        translations: [NodeLoopWhileTranslation!]!
    } 

    input NodeLoopWhileTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input NodeLoopWhileTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    type NodeLoopWhileTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }
`;
export const resolvers: {
} = {
};

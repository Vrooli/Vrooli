import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeLoopWhileCreateInput {
        id: ID!
        translationsCreate: [NodeLoopWhileTranslationCreateInput!]
        condition: String!
        toConnect: ID
    }
    input NodeLoopWhileUpdateInput {
        id: ID!
        toConnect: ID
        translationsDelete: [ID!]
        translationsCreate: [NodeLoopWhileTranslationCreateInput!]
        translationsUpdate: [NodeLoopWhileTranslationUpdateInput!]
        condition: String
    }
    type NodeLoopWhile {
        id: ID!
        toId: ID
        translations: [NodeLoopWhileTranslation!]!
        condition: String!
    } 

    input NodeLoopWhileTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String!
    }
    input NodeLoopWhileTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type NodeLoopWhileTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }
`
export const resolvers: {
} = {
}
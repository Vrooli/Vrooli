import { gql } from 'apollo-server-express';

export const typeDef = gql`
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
        name: String!
    }
    input NodeLinkWhenTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type NodeLinkWhenTranslation {
        id: ID!
        language: String!
        description: String
        name: String!
    }
`
export const resolvers: {
} = {
}
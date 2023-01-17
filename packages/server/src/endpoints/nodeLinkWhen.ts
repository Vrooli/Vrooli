import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeLinkWhenCreateInput {
        id: ID!
        linkConnect: ID!
        translationsCreate: [NodeLinkWhenTranslationCreateInput!]
        condition: String!
    }
    input NodeLinkWhenUpdateInput {
        id: ID!
        linkConnect: ID
        translationsDelete: [ID!]
        translationsCreate: [NodeLinkWhenTranslationCreateInput!]
        translationsUpdate: [NodeLinkWhenTranslationUpdateInput!]
        condition: String
    }
    type NodeLinkWhen {
        id: ID!
        translations: [NodeLinkWhenTranslation!]!
        condition: String!
        link: NodeLink!

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
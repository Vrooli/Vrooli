import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input RoutineVersionOutputCreateInput {
        id: ID!
        index: Int
        name: String
        routineVersionConnect: ID!
        standardVersionConnect: ID
        standardVersionCreate: StandardVersionCreateInput
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
    }
    input RoutineVersionOutputUpdateInput {
        id: ID!
        index: Int
        name: String
        standardVersionConnect: ID
        standardVersionDisconnect: Boolean
        standardVersionCreate: StandardVersionCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionOutputTranslationUpdateInput!]
    }
    type RoutineVersionOutput {
        id: ID!
        index: Int
        name: String
        routineVersion: RoutineVersion!
        standardVersion: StandardVersion
        translations: [RoutineVersionOutputTranslation!]!
    }

    input RoutineVersionOutputTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input RoutineVersionOutputTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type RoutineVersionOutputTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
`

const objectType = 'RoutineVersionOutput';
export const resolvers: {
} = {
}
import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input RoutineVersionOutputCreateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardVersionConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionOutputTranslationUpdateInput!]
    }
    input RoutineVersionOutputUpdateInput {
        id: ID!
        isRequired: Boolean
        name: String
        standardConnect: ID
        standardCreate: StandardCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionOutputTranslationUpdateInput!]
    }
    type RoutineVersionOutput {
        id: ID!
        isRequired: Boolean
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
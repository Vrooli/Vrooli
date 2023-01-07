import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input RoutineVersionOutputCreateInput {
        id: ID!
        index: Int
        isRequired: Boolean
        name: String
        routineVersionConnect: ID!
        standardVersionConnect: ID
        standardVersionCreate: StandardVersionCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionOutputTranslationUpdateInput!]
    }
    input RoutineVersionOutputUpdateInput {
        id: ID!
        index: Int
        isRequired: Boolean
        name: String
        standardVersionConnect: ID
        standardVersionDisconnect: ID
        standardVersionCreate: StandardVersionCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionOutputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionOutputTranslationUpdateInput!]
    }
    type RoutineVersionOutput {
        type: GqlModelType!
        id: ID!
        index: Int
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
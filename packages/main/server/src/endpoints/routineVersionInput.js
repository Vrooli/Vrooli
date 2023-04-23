import { gql } from "apollo-server-express";
export const typeDef = gql `
    input RoutineVersionInputCreateInput {
        id: ID!
        index: Int
        isRequired: Boolean
        name: String
        routineVersionConnect: ID!
        standardVersionConnect: ID
        standardVersionCreate: StandardVersionCreateInput
        translationsCreate: [RoutineVersionInputTranslationCreateInput!]
    }
    input RoutineVersionInputUpdateInput {
        id: ID!
        index: Int
        isRequired: Boolean
        name: String
        standardVersionConnect: ID
        standardVersionDisconnect: Boolean
        standardVersionCreate: StandardVersionCreateInput
        translationsDelete: [ID!]
        translationsCreate: [RoutineVersionInputTranslationCreateInput!]
        translationsUpdate: [RoutineVersionInputTranslationUpdateInput!]
    }
    type RoutineVersionInput {
        id: ID!
        index: Int
        isRequired: Boolean
        name: String
        routineVersion: RoutineVersion!
        standardVersion: StandardVersion
        translations: [RoutineVersionInputTranslation!]!
    }

    input RoutineVersionInputTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
    input RoutineVersionInputTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        helpText: String
    }
    type RoutineVersionInputTranslation {
        id: ID!
        language: String!
        description: String
        helpText: String
    }
`;
const objectType = "RoutineVersionInput";
export const resolvers = {};
//# sourceMappingURL=routineVersionInput.js.map
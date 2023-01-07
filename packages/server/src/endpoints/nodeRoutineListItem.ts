import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input NodeRoutineListItemCreateInput {
        id: ID!
        index: Int!
        isOptional: Boolean
        listConnect: ID!
        routineVersionConnect: ID!
        translationsCreate: [NodeRoutineListItemTranslationCreateInput!]
    }
    input NodeRoutineListItemUpdateInput {
        id: ID!
        index: Int
        isOptional: Boolean
        routineVersionUpdate: RoutineVersionUpdateInput
        translationsDelete: [ID!]
        translationsCreate: [NodeRoutineListItemTranslationCreateInput!]
        translationsUpdate: [NodeRoutineListItemTranslationUpdateInput!]
    }
    type NodeRoutineListItem {
        type: GqlModelType!
        id: ID!
        index: Int!
        isOptional: Boolean!
        routineVersion: RoutineVersion!
        translations: [NodeRoutineListItemTranslation!]!
    }

    input NodeRoutineListItemTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input NodeRoutineListItemTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type NodeRoutineListItemTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }
`
export const resolvers: {
} = {
}
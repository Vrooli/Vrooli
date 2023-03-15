import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input ProjectVersionDirectoryCreateInput {
        id: ID!
        childOrder: String
        isRoot: Boolean
        parentDirectoryConnect: ID
        projectVersionConnect: ID!
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
    }
    input ProjectVersionDirectoryUpdateInput {
        id: ID!
        childOrder: String
        isRoot: Boolean
        parentDirectoryConnect: ID
        parentDirectoryDisconnect: Boolean
        projectVersionConnect: ID
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
        translationsUpdate: [ProjectVersionDirectoryTranslationUpdateInput!]
        translationsDelete: [ID!]
    }
    type ProjectVersionDirectory {
        id: ID!
        created_at: Date!
        updated_at: Date!
        childOrder: String
        isRoot: Boolean!
        parentDirectory: ProjectVersionDirectory
        projectVersion: ProjectVersion
        children: [ProjectVersionDirectory!]!
        childApiVersions: [ApiVersion!]!
        childNoteVersions: [NoteVersion!]!
        childOrganizations: [Organization!]!
        childProjectVersions: [ProjectVersion!]!
        childRoutineVersions: [RoutineVersion!]!
        childSmartContractVersions: [SmartContractVersion!]!
        childStandardVersions: [StandardVersion!]!
        runProjectSteps: [RunProjectStep!]!
        translations: [ProjectVersionDirectoryTranslation!]!
    }

    input ProjectVersionDirectoryTranslationCreateInput {
        id: ID!
        language: String!
        description: String
        name: String
    }
    input ProjectVersionDirectoryTranslationUpdateInput {
        id: ID!
        language: String
        description: String
        name: String
    }
    type ProjectVersionDirectoryTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }
`

const objectType = 'ProjectVersionDirectory';
export const resolvers: {
} = {
}
import { gql } from 'apollo-server-express';

export const typeDef = gql`
    input ProjectVersionDirectoryCreateInput {
        id: ID!
        childApiVersionsConnect: [ID!]
        childNoteVersionsConnect: [ID!]
        childOrder: String
        childOrganizationsConnect: [ID!]
        childProjectVersionsConnect: [ID!]
        childRoutineVersionsConnect: [ID!]
        childSmartContractVersionsConnect: [ID!]
        childStandardVersionsConnect: [ID!]
        isRoot: Boolean
        parentDirectoryConnect: ID
        projectVersionConnect: ID!
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
    }
    input ProjectVersionDirectoryUpdateInput {
        id: ID!
        childApiVersionsConnect: [ID!]
        childApiVersionsDisconnect: [ID!]
        childNoteVersionsConnect: [ID!]
        childNoteVersionsDisconnect: [ID!]
        childOrder: String
        childOrganizationsConnect: [ID!]
        childOrganizationsDisconnect: [ID!]
        childProjectVersionsConnect: [ID!]
        childProjectVersionsDisconnect: [ID!]
        childRoutineVersionsConnect: [ID!]
        childRoutineVersionsDisconnect: [ID!]
        childSmartContractVersionsConnect: [ID!]
        childSmartContractVersionsDisconnect: [ID!]
        childStandardVersionsConnect: [ID!]
        childStandardVersionsDisconnect: [ID!]
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
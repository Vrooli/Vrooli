import { ProjectVersionDirectorySortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsProjectVersionDirectory, ProjectVersionDirectoryEndpoints } from "../logic/projectVersionDirectory";

export const typeDef = gql`
    enum ProjectVersionDirectorySortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input ProjectVersionDirectoryCreateInput {
        id: ID!
        childApiVersionsConnect: [ID!]
        childCodeVersionsConnect: [ID!]
        childNoteVersionsConnect: [ID!]
        childOrder: String
        childProjectVersionsConnect: [ID!]
        childRoutineVersionsConnect: [ID!]
        childStandardVersionsConnect: [ID!]
        childTeamsConnect: [ID!]
        isRoot: Boolean
        parentDirectoryConnect: ID
        projectVersionConnect: ID!
        translationsCreate: [ProjectVersionDirectoryTranslationCreateInput!]
    }
    input ProjectVersionDirectoryUpdateInput {
        id: ID!
        childApiVersionsConnect: [ID!]
        childApiVersionsDisconnect: [ID!]
        childCodeVersionsConnect: [ID!]
        childCodeVersionsDisconnect: [ID!]
        childNoteVersionsConnect: [ID!]
        childNoteVersionsDisconnect: [ID!]
        childOrder: String
        childProjectVersionsConnect: [ID!]
        childProjectVersionsDisconnect: [ID!]
        childRoutineVersionsConnect: [ID!]
        childRoutineVersionsDisconnect: [ID!]
        childStandardVersionsConnect: [ID!]
        childStandardVersionsDisconnect: [ID!]
        childTeamsConnect: [ID!]
        childTeamsDisconnect: [ID!]
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
        childCodeVersions: [CodeVersion!]!
        childNoteVersions: [NoteVersion!]!
        childProjectVersions: [ProjectVersion!]!
        childRoutineVersions: [RoutineVersion!]!
        childStandardVersions: [StandardVersion!]!
        childTeams: [Team!]!
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
        language: String!
        description: String
        name: String
    }
    type ProjectVersionDirectoryTranslation {
        id: ID!
        language: String!
        description: String
        name: String
    }

    input ProjectVersionDirectorySearchInput {
        after: String
        ids: [ID!]
        isRoot: Boolean
        parentDirectoryId: ID
        searchString: String
        sortBy: ProjectVersionDirectorySortBy
        take: Int
        visibility: VisibilityType
    }

    type ProjectVersionDirectorySearchResult {
        pageInfo: PageInfo!
        edges: [ProjectVersionDirectoryEdge!]!
    }

    type ProjectVersionDirectoryEdge {
        cursor: String!
        node: ProjectVersionDirectory!
    }

    extend type Query {
        projectVersionDirectory(input: FindByIdInput!): ProjectVersionDirectory
        projectVersionDirectories(input: ProjectVersionDirectorySearchInput!): ProjectVersionDirectorySearchResult!
    }

    extend type Mutation {
        projectVersionDirectoryCreate(input: ProjectVersionDirectoryCreateInput!): ProjectVersionDirectory!
        projectVersionDirectoryUpdate(input: ProjectVersionDirectoryUpdateInput!): ProjectVersionDirectory!
    }
`;

export const resolvers: {
    ProjectVersionDirectorySortBy: typeof ProjectVersionDirectorySortBy;
    Query: EndpointsProjectVersionDirectory["Query"];
} = {
    ProjectVersionDirectorySortBy,
    ...ProjectVersionDirectoryEndpoints,
};

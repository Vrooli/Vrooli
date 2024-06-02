import { LabelSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsLabel, LabelEndpoints } from "../logic/label";

export const typeDef = gql`
    enum LabelSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input LabelCreateInput {
        id: ID!
        label: String!
        color: String
        teamConnect: ID
        translationsCreate: [LabelTranslationCreateInput!]
    }
    input LabelUpdateInput {
        id: ID!
        label: String
        color: String
        apisConnect: [ID!]
        apisDisconnect: [ID!]
        codesConnect: [ID!]
        codesDisconnect: [ID!]
        issuesConnect: [ID!]
        issuesDisconnect: [ID!]
        meetingsConnect: [ID!]
        meetingsDisconnect: [ID!]
        notesConnect: [ID!]
        notesDisconnect: [ID!]
        projectsConnect: [ID!]
        projectsDisconnect: [ID!]
        routinesConnect: [ID!]
        routinesDisconnect: [ID!]
        schedulesConnect: [ID!]
        schedulesDisconnect: [ID!]
        standardsConnect: [ID!]
        standardsDisconnect: [ID!]
        focusModesConnect: [ID!]
        focusModesDisconnect: [ID!]
        translationsDelete: [ID!]
        translationsCreate: [LabelTranslationCreateInput!]
        translationsUpdate: [LabelTranslationUpdateInput!]
    }
    type Label {
        id: ID!
        created_at: Date!
        updated_at: Date!
        label: String!
        color: String
        apis: [Api!]
        apisCount: Int!
        codes: [Code!]
        codesCount: Int!
        focusModes: [FocusMode!]
        focusModesCount: Int!
        issues: [Issue!]
        issuesCount: Int!
        meetings: [Meeting!]
        meetingsCount: Int!
        notes: [Note!]
        notesCount: Int!
        owner: Owner!
        projects: [Project!]
        projectsCount: Int!
        routines: [Routine!]
        routinesCount: Int!
        schedules: [Schedule!]
        schedulesCount: Int!
        standards: [Standard!]
        standardsCount: Int!
        translations: [LabelTranslation!]!
        translationsCount: Int!
        you: LabelYou!
    }

    type LabelYou {
        canDelete: Boolean!
        canUpdate: Boolean!
    }

    input LabelTranslationCreateInput {
        id: ID!
        language: String!
        description: String!
    }
    input LabelTranslationUpdateInput {
        id: ID!
        language: String!
        description: String
    }
    type LabelTranslation {
        id: ID!
        language: String!
        description: String!
    }

    input LabelSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        ownedByTeamId: ID
        ownedByUserId: ID
        searchString: String
        sortBy: LabelSortBy
        take: Int
        translationLanguages: [String!]
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type LabelSearchResult {
        pageInfo: PageInfo!
        edges: [LabelEdge!]!
    }

    type LabelEdge {
        cursor: String!
        node: Label!
    }

    extend type Query {
        label(input: FindByIdInput!): Label
        labels(input: LabelSearchInput!): LabelSearchResult!
    }

    extend type Mutation {
        labelCreate(input: LabelCreateInput!): Label!
        labelUpdate(input: LabelUpdateInput!): Label!
    }
`;

export const resolvers: {
    LabelSortBy: typeof LabelSortBy;
    Query: EndpointsLabel["Query"];
    Mutation: EndpointsLabel["Mutation"];
} = {
    LabelSortBy,
    ...LabelEndpoints,
};

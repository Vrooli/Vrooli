import { FindByIdInput, Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';
import { rateLimit } from '../middleware';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';

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
        organizationConnect: ID
        translationsCreate: [LabelTranslationCreateInput!]
    }
    input LabelUpdateInput {
        id: ID!
        label: String
        color: String
        apisConnect: [ID!]
        apisDisconnect: [ID!]
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
        smartContractsConnect: [ID!]
        smartContractsDisconnect: [ID!]
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
        smartContracts: [SmartContract!]
        smartContractsCount: Int!
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
        language: String
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
        ownedByOrganizationId: ID
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
`

const objectType = 'Label';
export const resolvers: {
    LabelSortBy: typeof LabelSortBy;
    Query: {
        label: GQLEndpoint<FindByIdInput, FindOneResult<Label>>;
        labels: GQLEndpoint<LabelSearchInput, FindManyResult<Label>>;
    },
    Mutation: {
        labelCreate: GQLEndpoint<LabelCreateInput, CreateOneResult<Label>>;
        labelUpdate: GQLEndpoint<LabelUpdateInput, UpdateOneResult<Label>>;
    }
} = {
    LabelSortBy: LabelSortBy,
    Query: {
        label: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        labels: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        labelCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        labelUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
import { gql } from 'apollo-server-express';
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { FindByIdInput, Label, LabelSearchInput, LabelCreateInput, LabelUpdateInput, ReportResponseSortBy, ReportResponseSearchInput, ReportResponse, ReportResponseCreateInput, ReportResponseUpdateInput } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, readManyHelper, readOneHelper, updateHelper } from '../actions';

export const typeDef = gql`
    enum ReportResponseSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    enum ReportSuggestedAction {
        Delete
        FalseReport
        HideUntilFixed
        NonIssue
        SuspendUser
    }

    input ReportResponseCreateInput {
        id: ID!
        reportConnect: ID!
        actionSuggested: ReportSuggestedAction!
        details: String
        language: String
    }
    input ReportResponseUpdateInput {
        id: ID!
        actionSuggested: ReportSuggestedAction
        details: String
        language: String
    }
    type ReportResponse {
        id: ID!
        created_at: Date!
        updated_at: Date!
        actionSuggested: ReportSuggestedAction!
        details: String
        language: String
        report: Report!
        permissionsReportResponse: ReportResponsePermission!
    }

    type ReportResponsePermission {
        canDelete: Boolean!
        canEdit: Boolean!
    }

    input ReportResponseSearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        languageIn: [String!]
        reportId: ID
        searchString: String
        sortBy: ReportResponseSortBy
        take: Int
        updatedTimeFrame: TimeFrame
        userId: ID
        visibility: VisibilityType
    }

    type ReportResponseSearchResult {
        pageInfo: PageInfo!
        edges: [ReportResponseEdge!]!
    }

    type ReportResponseEdge {
        cursor: String!
        node: ReportResponse!
    }

    extend type Query {
        reportResponse(input: FindByIdInput!): ReportResponse
        reportResponses(input: ReportResponseSearchInput!): ReportResponseSearchResult!
    }

    extend type Mutation {
        reportResponseCreate(input: ReportResponseCreateInput!): ReportResponse!
        reportResponseUpdate(input: ReportResponseUpdateInput!): ReportResponse!
    }
`

const objectType = 'ReportResponse';
export const resolvers: {
    ReportResponseSortBy: typeof ReportResponseSortBy;
    Query: {
        reportResponse: GQLEndpoint<FindByIdInput, FindOneResult<ReportResponse>>;
        reportResponses: GQLEndpoint<ReportResponseSearchInput, FindManyResult<ReportResponse>>;
    },
    Mutation: {
        reportResponseCreate: GQLEndpoint<ReportResponseCreateInput, CreateOneResult<ReportResponse>>;
        reportResponseUpdate: GQLEndpoint<ReportResponseUpdateInput, UpdateOneResult<ReportResponse>>;
    }
} = {
    ReportResponseSortBy,
    Query: {
        reportResponse: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        reportResponses: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
    Mutation: {
        reportResponseCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        reportResponseUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
    }
}
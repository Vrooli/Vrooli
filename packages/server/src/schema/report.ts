import { gql } from 'apollo-server-express';
import { FindByIdInput, Report, ReportCountInput, ReportCreateInput, ReportFor, ReportSearchInput, ReportSearchResult, ReportSortBy, ReportUpdateInput } from './types';
import { IWrap, RecursivePartial } from '../types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, readManyHelper, readOneHelper, ReportModel, updateHelper } from '../models';

export const typeDef = gql`
    enum ReportFor {
        Comment
        Organization
        Project
        Routine
        Standard
        Tag
        User
    }   

    enum ReportSortBy {
        DateCreatedAsc
        DateCreatedDesc
    }

    input ReportCreateInput {
        id: ID!
        createdFor: ReportFor!
        createdForId: ID!
        details: String
        language: String!
        reason: String!
    }
    input ReportUpdateInput {
        id: ID!
        details: String
        language: String
        reason: String
    }

    type Report {
        id: ID
        isOwn: Boolean!
        details: String
        language: String!
        reason: String!
    }

    input ReportSearchInput {
        userId: ID
        organizationId: ID
        projectId: ID
        routineId: ID
        standardId: ID
        tagId: ID
        ids: [ID!]
        languages: [String!]
        sortBy: ReportSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
    }

    # Return type for search result
    type ReportSearchResult {
        pageInfo: PageInfo!
        edges: [ReportEdge!]!
    }

    # Return type for search result edge
    type ReportEdge {
        cursor: String!
        node: Report!
    }

    # Input for count
    input ReportCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        report(input: FindByIdInput!): Report
        reports(input: ReportSearchInput!): ReportSearchResult!
        reportsCount(input: ReportCountInput!): Int!
    }

    extend type Mutation {
        reportCreate(input: ReportCreateInput!): Report!
        reportUpdate(input: ReportUpdateInput!): Report!
    }
`

export const resolvers = {
    ReportFor: ReportFor,
    ReportSortBy: ReportSortBy,
    Query: {
        report: async (_parent: undefined, { input }: IWrap<FindByIdInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report> | null> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, model: ReportModel, prisma, req })
        },
        reports: async (_parent: undefined, { input }: IWrap<ReportSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ReportSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, model: ReportModel, prisma, req })
        },
        reportsCount: async (_parent: undefined, { input }: IWrap<ReportCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, model: ReportModel, prisma, req })
        },
    },
    Mutation: {
        reportCreate: async (_parent: undefined, { input }: IWrap<ReportCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, model: ReportModel, prisma, req })
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, model: ReportModel, prisma, req })
        },
    }
}
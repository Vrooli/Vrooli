import { gql } from 'apollo-server-express';
import { FindByIdInput, Report, ReportCountInput, ReportCreateInput, ReportFor, ReportSearchInput, ReportSearchResult, ReportSortBy, ReportUpdateInput } from './types';
import { IWrap, RecursivePartial } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { countHelper, createHelper, readManyHelper, readOneHelper, ReportModel, updateHelper } from '../models';
import { rateLimit } from '../rateLimit';

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
        DateUpdatedAsc
        DateUpdatedDesc
    }

    input ReportCreateInput {
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
        report: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, ReportModel(context.prisma));
        },
        reports: async (_parent: undefined, { input }: IWrap<ReportSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<ReportSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, ReportModel(context.prisma));
        },
        reportsCount: async (_parent: undefined, { input }: IWrap<ReportCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, ReportModel(context.prisma));
        },
    },
    Mutation: {
        reportCreate: async (_parent: undefined, { input }: IWrap<ReportCreateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            await rateLimit({ context, info, max: 500, byAccount: true });
            return createHelper(context.req.userId, input, info, ReportModel(context.prisma));
        },
        reportUpdate: async (_parent: undefined, { input }: IWrap<ReportUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Report>> => {
            await rateLimit({ context, info, max: 1000, byAccount: true });
            return updateHelper(context.req.userId, input, info, ReportModel(context.prisma));
        },
    }
}
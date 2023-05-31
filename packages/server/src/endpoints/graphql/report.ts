import { FindByIdInput, Report, ReportCreateInput, ReportFor, ReportSearchInput, ReportSortBy, ReportUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { rateLimit } from "../../middleware";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export const typeDef = gql`
    enum ReportFor {
        ApiVersion
        Comment
        Issue
        NoteVersion
        Organization
        Post
        ProjectVersion
        RoutineVersion
        StandardVersion
        Tag
        User
    }   

    enum ReportSortBy {
        DateCreatedAsc
        DateCreatedDesc
        ResponsesAsc
        ResponsesDesc
    }

    input ReportCreateInput {
        id: ID!
        createdFor: ReportFor!
        createdForConnect: ID!
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
        id: ID!
        created_at: Date!
        updated_at: Date!
        details: String
        language: String!
        reason: String!
        responses: [ReportResponse!]!
        responsesCount: Int!
        you: ReportYou!
    }

    type ReportYou {
        canDelete: Boolean!
        canRespond: Boolean!
        canUpdate: Boolean!
    }

    input ReportSearchInput {
        ids: [ID!]
        languageIn: [String!]
        sortBy: ReportSortBy
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        searchString: String
        after: String
        take: Int
        fromId: ID
        apiVersionId: ID
        commentId: ID
        issueId: ID
        noteVersionId: ID
        organizationId: ID
        postId: ID
        projectVersionId: ID
        routineVersionId: ID
        smartContractVersionId: ID
        standardVersionId: ID
        tagId: ID
        userId: ID
    }

    type ReportSearchResult {
        pageInfo: PageInfo!
        edges: [ReportEdge!]!
    }

    type ReportEdge {
        cursor: String!
        node: Report!
    }

    extend type Query {
        report(input: FindByIdInput!): Report
        reports(input: ReportSearchInput!): ReportSearchResult!
    }

    extend type Mutation {
        reportCreate(input: ReportCreateInput!): Report!
        reportUpdate(input: ReportUpdateInput!): Report!
    }
`;

const objectType = "Report";
export const resolvers: {
    ReportFor: typeof ReportFor;
    ReportSortBy: typeof ReportSortBy;
    Query: {
        report: GQLEndpoint<FindByIdInput, FindOneResult<Report>>;
        reports: GQLEndpoint<ReportSearchInput, FindManyResult<Report>>;
    },
    Mutation: {
        reportCreate: GQLEndpoint<ReportCreateInput, CreateOneResult<Report>>;
        reportUpdate: GQLEndpoint<ReportUpdateInput, UpdateOneResult<Report>>;
    }
} = {
    ReportFor,
    ReportSortBy,
    Query: {
        report: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        reports: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        reportCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 500, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        reportUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

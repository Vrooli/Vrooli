import { ReportResponseSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsReportResponse, ReportResponseEndpoints } from "../logic/reportResponse";

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
        you: ReportResponseYou!
    }

    type ReportResponseYou {
        canDelete: Boolean!
        canUpdate: Boolean!
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
`;

export const resolvers: {
    ReportResponseSortBy: typeof ReportResponseSortBy;
    Query: EndpointsReportResponse["Query"];
    Mutation: EndpointsReportResponse["Mutation"];
} = {
    ReportResponseSortBy,
    ...ReportResponseEndpoints,
};

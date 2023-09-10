import { ReportFor, ReportSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsReport, ReportEndpoints } from "../logic";

export const typeDef = gql`
    enum ReportFor {
        ApiVersion
        ChatMessage
        Comment
        Issue
        NoteVersion
        Organization
        Post
        ProjectVersion
        RoutineVersion
        SmartContractVersion
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
        chatMessageId: ID
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

export const resolvers: {
    ReportFor: typeof ReportFor;
    ReportSortBy: typeof ReportSortBy;
    Query: EndpointsReport["Query"];
    Mutation: EndpointsReport["Mutation"];
} = {
    ReportFor,
    ReportSortBy,
    ...ReportEndpoints,
};

import { ReputationHistorySortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { EndpointsReputationHistory, ReputationHistoryEndpoints } from "../logic";

export const typeDef = gql`
    enum ReputationHistorySortBy {
        AmountAsc
        AmountDesc
        DateCreatedAsc
        DateCreatedDesc
    }

    type ReputationHistory {
        id: ID!
        created_at: Date!
        updated_at: Date!
        amount: Int!
        event: String!
        objectId1: ID
        objectId2: ID
    }

    input ReputationHistorySearchInput {
        after: String
        createdTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: ReputationHistorySortBy
        tags: [String!]
        take: Int
        updatedTimeFrame: TimeFrame
        visibility: VisibilityType
    }

    type ReputationHistorySearchResult {
        pageInfo: PageInfo!
        edges: [ReputationHistoryEdge!]!
    }

    type ReputationHistoryEdge {
        cursor: String!
        node: ReputationHistory!
    }

    extend type Query {
        reputationHistory(input: FindByIdInput!): ReputationHistory
        reputationHistories(input: ReputationHistorySearchInput!): ReputationHistorySearchResult!
    }
`;

export const resolvers: {
    ReputationHistorySortBy: typeof ReputationHistorySortBy;
    Query: EndpointsReputationHistory["Query"];
} = {
    ReputationHistorySortBy,
    ...ReputationHistoryEndpoints,
};

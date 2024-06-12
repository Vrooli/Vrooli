import { ViewSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { UnionResolver } from "../../types";
import { EndpointsView, ViewEndpoints } from "../logic/view";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
    enum ViewSortBy {
        LastViewedAsc
        LastViewedDesc
    }

    union ViewTo = Api | Code | Issue | Note | Post | Project | Question | Routine | Standard | Team | User
  
    type View {
        id: ID!
        by: User!
        lastViewedAt: Date!
        name: String!
        to: ViewTo!
    }

    input ViewSearchInput {
        after: String
        lastViewedTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: ViewSortBy
        take: Int
    }

    type ViewSearchResult {
        pageInfo: PageInfo!
        edges: [ViewEdge!]!
    }

    type ViewEdge {
        cursor: String!
        node: View!
    }

    extend type Query {
        views(input: ViewSearchInput!): ViewSearchResult!
    }
`;

export const resolvers: {
    ViewSortBy: typeof ViewSortBy;
    ViewTo: UnionResolver;
    Query: EndpointsView["Query"];
} = {
    ViewSortBy,
    ViewTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    ...ViewEndpoints,
};

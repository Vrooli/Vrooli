import { ViewSortBy } from "@local/consts";
import { gql } from "apollo-server-express";
import { readManyHelper } from "../actions";
import { assertRequestFrom } from "../auth/request";
import { rateLimit } from "../middleware";
import { resolveUnion } from "./resolvers";
export const typeDef = gql `
    enum ViewSortBy {
        LastViewedAsc
        LastViewedDesc
    }

    union ViewTo = Api | Issue | Note | Organization | Post | Project | Question | Routine | SmartContract | Standard | User
  
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
const objectType = "View";
export const resolvers = {
    ViewSortBy,
    ViewTo: { __resolveType(obj) { return resolveUnion(obj); } },
    Query: {
        views: async (_parent, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { byId: userData.id } });
        },
    },
};
//# sourceMappingURL=view.js.map
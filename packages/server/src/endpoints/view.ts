import { View, ViewSearchInput, ViewSearchResult, ViewSortBy } from "@local/shared";
import { gql } from "apollo-server-express";
import { GraphQLResolveInfo } from "graphql";
import { readManyHelper } from "../actions";
import { assertRequestFrom } from "../auth/request";
import { Context, rateLimit } from "../middleware";
import { FindManyResult, GQLEndpoint, IWrap, UnionResolver } from "../types";
import { resolveUnion } from "./resolvers";

export const typeDef = gql`
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
export const resolvers: {
    ViewSortBy: typeof ViewSortBy;
    ViewTo: UnionResolver;
    Query: {
        views: GQLEndpoint<ViewSearchInput, FindManyResult<View>>;
    },
} = {
    ViewSortBy,
    ViewTo: { __resolveType(obj: any) { return resolveUnion(obj); } },
    Query: {
        views: async (_parent: undefined, { input }: IWrap<ViewSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ViewSearchResult> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { byId: userData.id } });
        },
    },
};

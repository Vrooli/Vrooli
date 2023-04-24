import { FindByIdInput, Payment, PaymentSearchInput, PaymentSortBy, PaymentStatus, PaymentType } from "@local/shared";
import { gql } from "apollo-server-express";
import { readManyHelper, readOneHelper } from "../actions";
import { rateLimit } from "../middleware";
import { FindManyResult, FindOneResult, GQLEndpoint } from "../types";

export const typeDef = gql`
    enum PaymentSortBy {
        AmountAsc
        AmountDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
    }

    enum PaymentStatus {
        Pending
        Paid
        Failed
    }

    enum PaymentType {
        PremiumMonthly
        PremiumYearly
        Donation
    }


    type Payment {
        id: ID!
        created_at: Date!
        updated_at: Date!
        amount: Int!
        checkoutId: String!
        currency: String!
        description: String!
        status: PaymentStatus!
        paymentMethod: String!
        paymentType: PaymentType!
        cardType: String
        cardExpDate: String
        cardLast4: String
        organization: Organization!
        user: User!
    }

    input PaymentSearchInput {
        after: String
        cardLast4: String
        createdTimeFrame: TimeFrame
        currency: String
        ids: [ID!]
        maxAmount: Int
        minAmount: Int
        sortBy: PaymentSortBy
        status: PaymentStatus
        take: Int
        updatedTimeFrame: TimeFrame
    }

    type PaymentSearchResult {
        pageInfo: PageInfo!
        edges: [PaymentEdge!]!
    }

    type PaymentEdge {
        cursor: String!
        node: Payment!
    }

    extend type Query {
        payment(input: FindByIdInput!): Payment
        payments(input: PaymentSearchInput!): PaymentSearchResult!
    }
`;

const objectType = "Payment";
export const resolvers: {
    PaymentSortBy: typeof PaymentSortBy;
    PaymentStatus: typeof PaymentStatus;
    PaymentType: typeof PaymentType;
    Query: {
        payment: GQLEndpoint<FindByIdInput, FindOneResult<Payment>>;
        payments: GQLEndpoint<PaymentSearchInput, FindManyResult<Payment>>;
    },
} = {
    PaymentSortBy,
    PaymentStatus,
    PaymentType,
    Query: {
        payment: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        payments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
};

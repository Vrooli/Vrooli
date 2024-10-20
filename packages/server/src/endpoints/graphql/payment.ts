import { PaymentSortBy, PaymentStatus, PaymentType } from "@local/shared";
import { EndpointsPayment, PaymentEndpoints } from "../logic/payment";

export const typeDef = `#graphql
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
        Credits
        Donation
        PremiumMonthly
        PremiumYearly
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
        team: Team!
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

export const resolvers: {
    PaymentSortBy: typeof PaymentSortBy;
    PaymentStatus: typeof PaymentStatus;
    PaymentType: typeof PaymentType;
    Query: EndpointsPayment["Query"];
} = {
    PaymentSortBy,
    PaymentStatus,
    PaymentType,
    ...PaymentEndpoints,
};

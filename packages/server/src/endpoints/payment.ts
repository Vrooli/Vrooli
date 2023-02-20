import { gql } from 'apollo-server-express';
import { FindByIdInput, PaymentSortBy, PaymentStatus } from '@shared/consts';
import { FindManyResult, FindOneResult, GQLEndpoint } from '../types';
import { rateLimit } from '../middleware';
import { readManyHelper, readOneHelper } from '../actions';

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

    type Payment {
        id: ID!
        created_at: Date!
        updated_at: Date!
        amount: Int!
        currency: String!
        description: String!
        status: PaymentStatus!
        paymentMethod: String!
        cardType: String
        cardExpDate: String
        cardLast4: String
        organization: Organization!
        user: User!
    }

    input PaymentSearchInput {
        after: String
        cardType: String
        cardExpDate: String
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
        visibility: VisibilityType
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
`

const objectType = 'Payment';
export const resolvers: {
    PaymentSortBy: typeof PaymentSortBy;
    PaymentStatus: typeof PaymentStatus;
    Query: {
        payment: GQLEndpoint<FindByIdInput, FindOneResult<any>>;
        payments: GQLEndpoint<any, FindManyResult<any>>;
    },
} = {
    PaymentSortBy,
    PaymentStatus,
    Query: {
        payment: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req })
        },
        payments: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req })
        },
    },
}
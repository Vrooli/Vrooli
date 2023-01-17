import { gql } from 'apollo-server-express';
import { PaymentStatus } from '@shared/consts';

export const typeDef = gql`
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
`

export const resolvers: {
    PaymentStatus: typeof PaymentStatus;
} = {
    PaymentStatus,
}
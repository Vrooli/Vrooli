import { gql } from 'apollo-server-express';
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { Phone, PhoneCreateInput, PhoneUpdateInput, SendVerificationTextInput, Success } from './types';
import { rateLimit } from '../middleware';
import { createHelper, updateHelper } from '../actions';
import { setupVerificationCode } from '../auth';

export const typeDef = gql`
    input PhoneCreateInput {
        emailAddress: String!
        receivesAccountUpdates: Boolean
        receivesBusinessUpdates: Boolean
    }
    input PhoneUpdateInput {
        id: ID!
        receivesAccountUpdates: Boolean
        receivesBusinessUpdates: Boolean
    }
    input SendVerificationTextInput {
        phoneNumber: String!
    }

    type Phone {
        id: ID!
        emailAddress: String!
        receivesAccountUpdates: Boolean!
        receivesBusinessUpdates: Boolean!
        verified: Boolean!
        userId: ID
        user: User
    }

    extend type Mutation {
        phoneCreate(input: PhoneCreateInput!): Phone!
        phoneUpdate(input: PhoneUpdateInput!): Phone!
        sendVerificationText(input: SendVerificationTextInput!): Success!
    }
`

const objectType = 'Phone';
export const resolvers: {
    Mutation: {
        phoneCreate: GQLEndpoint<PhoneCreateInput, CreateOneResult<Phone>>;
        phoneUpdate: GQLEndpoint<PhoneUpdateInput, UpdateOneResult<Phone>>;
        sendVerificationText: GQLEndpoint<SendVerificationTextInput, Success>;
    }
} = {
    Mutation: {
        phoneCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        phoneUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        sendVerificationText: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            await setupVerificationCode(input.phoneNumber, prisma, req.languages);
            return { success: true };   
        },
    }
}
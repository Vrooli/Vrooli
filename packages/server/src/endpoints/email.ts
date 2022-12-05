import { gql } from 'apollo-server-express';
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { Email, EmailCreateInput, EmailUpdateInput, SendVerificationEmailInput, Success } from './types';
import { rateLimit } from '../middleware';
import { createHelper, updateHelper } from '../actions';
import { setupVerificationCode } from '../auth';

export const typeDef = gql`
    input EmailCreateInput {
        emailAddress: String!
        receivesAccountUpdates: Boolean
        receivesBusinessUpdates: Boolean
    }
    input EmailUpdateInput {
        id: ID!
        receivesAccountUpdates: Boolean
        receivesBusinessUpdates: Boolean
    }
    input SendVerificationEmailInput {
        emailAddress: String!
    }

    type Email {
        id: ID!
        emailAddress: String!
        receivesAccountUpdates: Boolean!
        receivesBusinessUpdates: Boolean!
        verified: Boolean!
        userId: ID
        user: User
    }

    extend type Mutation {
        emailCreate(input: EmailCreateInput!): Email!
        emailUpdate(input: EmailUpdateInput!): Email!
        sendVerificationEmail(input: SendVerificationEmailInput!): Success!
    }
`

const objectType = 'Email';
export const resolvers: {
    Mutation: {
        emailCreate: GQLEndpoint<EmailCreateInput, CreateOneResult<Email>>;
        emailUpdate: GQLEndpoint<EmailUpdateInput, UpdateOneResult<Email>>;
        sendVerificationEmail: GQLEndpoint<SendVerificationEmailInput, Success>;
    }
} = {
    Mutation: {
        emailCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        emailUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        sendVerificationEmail: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 50, req });
            await setupVerificationCode(input.emailAddress, prisma, req.languages);
            return { success: true };   
        },
    }
}
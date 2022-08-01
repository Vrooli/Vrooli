import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from '../types';
import { Email, EmailCreateInput, EmailUpdateInput, Success } from './types';
import { Context } from '../context';
import { createHelper, EmailModel, ProfileModel, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';

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

export const resolvers = {
    Mutation: {
        /**
         * Associate a new email address to your account.
         */
        emailCreate: async (_parent: undefined, { input }: IWrap<EmailCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            await rateLimit({ info, max: 10, byAccountOrKey: true, req });
            return createHelper({ info, input, model: EmailModel, prisma, userId: req.userId })
        },
        /**
         * Update an existing email address that is associated with your account.
         */
        emailUpdate: async (_parent: undefined, { input }: IWrap<EmailUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            await rateLimit({ info, max: 10, byAccountOrKey: true, req });
            return updateHelper({ info, input, model: EmailModel, prisma, userId: req.userId })
        },
        sendVerificationEmail: async (_parent: undefined, { input }: IWrap<any>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ info, max: 50, byAccountOrKey: true, req });
            await ProfileModel.verify.setupVerificationCode(input.emailAddress, prisma);
            return { success: true };   
        },
    }
}
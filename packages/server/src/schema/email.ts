import { gql } from 'apollo-server-express';
import { IWrap, RecursivePartial } from 'types';
import { DeleteOneInput, Email, EmailCreateInput, EmailUpdateInput, Success } from './types';
import { Context } from '../context';
import { createHelper, deleteOneHelper, EmailModel, ProfileModel, updateHelper } from '../models';
import { GraphQLResolveInfo } from 'graphql';

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
        emailDeleteOne(input: DeleteOneInput!): Success!
        sendVerificationEmail(input: SendVerificationEmailInput!): Success!
    }
`

export const resolvers = {
    Mutation: {
        /**
         * Associate a new email address to your account.
         */
        emailCreate: async (_parent: undefined, { input }: IWrap<EmailCreateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            return createHelper(req.userId, input, info, EmailModel(prisma));
        },
        /**
         * Update an existing email address that is associated with your account.
         */
        emailUpdate: async (_parent: undefined, { input }: IWrap<EmailUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            return updateHelper(req.userId, input, info, EmailModel(prisma));
        },
        emailDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            return deleteOneHelper(req.userId, input, EmailModel(prisma));
        },
        sendVerificationEmail: async (_parent: undefined, { input }: IWrap<any>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            ProfileModel(prisma).setupVerificationCode(input.emailAddress, prisma);
            return { success: true };   
        },
    }
}
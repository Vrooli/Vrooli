import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from 'types';
import { Count, DeleteManyInput, Email, EmailInput } from './types';
import { Context } from '../context';
import { emailFormatter, EmailModel } from '../models';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    input EmailInput {
        id: ID
        emailAddress: String!
        receivesAccountUpdates: Boolean
        receivesBusinessUpdates: Boolean
        userId: ID
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
        emailAdd(input: EmailInput!): Email!
        emailUpdate(input: EmailInput!): Email!
        emailDeleteMany(input: DeleteManyInput!): Count!
    }
`

export const resolvers = {
    Mutation: {
        /**
         * Associate a new email address to your account.
         */
        emailAdd: async (_parent: undefined, { input }: IWrap<EmailInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            // Must be adding to your own
            if(req.userId !== input.userId) throw new CustomError(CODE.Unauthorized);
            // Create object
            const dbModel = await EmailModel(prisma).create(input, info);
            // Format object to GraphQL type
            return emailFormatter().toGraphQL(dbModel);
        },
        /**
         * Update an existing email address that is associated with your account.
         */
        emailUpdate: async (_parent: undefined, { input }: IWrap<EmailInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Email>> => {
            // Find the email object in the database
            const curr = await EmailModel(prisma).findById({ id: input.id as string }, { userId: true });
            // Validate that the email belongs to the user
            if (curr === null || req.userId !== curr.userId) throw new CustomError(CODE.Unauthorized);
            // Update object
            const dbModel = await EmailModel(prisma).update(input, info);
            // Format to GraphQL type
            return emailFormatter().toGraphQL(dbModel);
        },
        emailDeleteMany: async (_parent: undefined, { input }: IWrap<DeleteManyInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Count> => {
            // Must deleting your own
            // TODO must keep at least one email per user
            const specified = await prisma.email.findMany({ where: { id: { in: input.ids } } });
            if (!specified) throw new CustomError(CODE.ErrorUnknown);
            const userIds = [...new Set(specified.map((s: any) => s.userId))];
            if (userIds.length > 1 || req.userId !== userIds[0]) throw new CustomError(CODE.Unauthorized);
            return await prisma.email.deleteMany({ where: { id: { in: input.ids } } });
        }
    }
}
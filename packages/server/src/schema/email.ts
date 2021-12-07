import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input EmailInput {
        id: ID
        emailAddress: String!
        receivesDeliveryUpdates: Boolean
        userId: ID
    }

    type Email {
        id: ID!
        emailAddress: String!
        receivesDeliveryUpdates: Boolean!
        verified: Boolean!
        user: User
    }

    extend type Mutation {
        addEmail(input: EmailInput!): Email!
        updateEmail(input: EmailInput!): Email!
        deleteEmails(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Mutation: {
        addEmail: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be adding to your own
            if(context.req.userId !== args.input.userId) return new CustomError(CODE.Unauthorized);
            return await context.prisma.email.create((new PrismaSelect(info).value), { data: { ...args.input } });
        },
        updateEmail: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be updating your own
            const curr = await context.prisma.email.findUnique({ where: { id: args.input.id } });
            if (context.req.userId !== curr.userId) return new CustomError(CODE.Unauthorized);
            return await context.prisma.email.update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteEmails: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must deleting your own
            // TODO must keep at least one email per user
            const specified = await context.prisma.email.findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const userIds = [...new Set(specified.map((s: any) => s.userId))];
            if (userIds.length > 1 || context.req.userId !== userIds[0]) return new CustomError(CODE.Unauthorized);
            return await context.prisma.email.deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
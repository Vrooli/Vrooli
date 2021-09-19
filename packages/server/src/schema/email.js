import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Email;

export const typeDef = gql`
    input EmailInput {
        id: ID
        emailAddress: String!
        receivesDeliveryUpdates: Boolean
        customerId: ID
        businessId: ID
    }

    type Email {
        id: ID!
        emailAddress: String!
        receivesDeliveryUpdates: Boolean!
        customer: Customer
        business: Business
    }

    extend type Query {
        emails: [Email!]!
    }

    extend type Mutation {
        addEmail(input: EmailInput!): Email!
        updateEmail(input: EmailInput!): Email!
        deleteEmails(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        emails: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addEmail: async (_, args, context, info) => {
            // Must be admin, or adding to your own
            if(!context.req.isAdmin || (context.req.businessId !== args.input.businessId)) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } });
        },
        updateEmail: async (_, args, context, info) => {
            // Must be admin, or updating your own
            if(!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            const curr = await context.prisma[_model].findUnique({ where: { id: args.input.id } });
            if (context.req.businessId !== curr.businessId) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteEmails: async (_, args, context) => {
            // Must be admin, or deleting your own
            // TODO must keep at least one email per customer
            const specified = await context.prisma[_model].findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const business_ids = [...new Set(specified.map(s => s.businessId))];
            if (!context.req.isAdmin && (business_ids.length > 1 || context.req.business_id !== business_ids[0])) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
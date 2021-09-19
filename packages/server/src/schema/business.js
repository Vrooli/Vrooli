import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { TABLES } from '../db';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Business;

export const typeDef = gql`
    input BusinessInput {
        id: ID
        name: String!
        subscribedToNewsletters: Boolean
        discountIds: [ID!]
        employeeIds: [ID!]
    }

    type Business {
        id: ID!
        name: String!
        subscribedToNewsletters: Boolean!
        addresses: [Address!]!
        phones: [Phone!]!
        emails: [Email!]!
        employees: [Customer!]!
        discounts: [Discount!]!
    }

    extend type Query {
        businesses: [Business!]!
    }

    extend type Mutation {
        addBusiness(input: BusinessInput!): Business!
        updateBusiness(input: BusinessInput!): Business!
        deleteBusinesses(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        businesses: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addBusiness: async(_, args, context, info) => {
            // Must be admin
            if(!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        updateBusiness: async(_, args, context, info) => {
            // Must be admin, or updating your own
            if(!context.req.isAdmin || (context.req.businessId !== args.input.id)) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteBusinesses: async(_, args, context) => {
            // Must be admin, or deleting your own
            if(!context.req.isAdmin || args.ids.length > 1 || context.req.businessId !== args.ids[0]) return new CustomError(CODE.Unauthorized); 
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        },
    }
}
import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { TABLES } from '../db';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Discount;

export const typeDef = gql`
    input DiscountInput {
        id: ID
        title: String!
        discount: Float!
        comment: String
        terms: String
        businessIds: [ID!]
        skuIds: [ID!]
    }

    type Discount {
        id: ID!
        discount: Float!
        title: String!
        comment: String
        terms: String
        businesses: [Business!]!
        skus: [Sku!]!
    }

    extend type Query {
        discounts: [Discount!]!
    }

    extend type Mutation {
        addDiscount(input: DiscountInput!): Discount!
        updateDiscount(input: DiscountInput!): Discount!
        deleteDiscounts(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        discounts: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addDiscount: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        updateDiscount: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteDiscounts: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
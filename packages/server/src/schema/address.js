import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Address;

export const typeDef = gql`
    input AddressInput {
        id: ID
        tag: String
        name: String
        country: String!
        administrativeArea: String!
        subAdministrativeArea: String
        locality: String!
        postalCode: String!
        throughfare: String!
        premise: String
        deliveryInstructions: String
        businessId: ID!
    }

    type Address {
        id: ID!
        tag: String
        name: String
        country: String!
        administrativeArea: String!
        subAdministrativeArea: String
        locality: String!
        postalCode: String!
        throughfare: String!
        premise: String
        business: Business!
        orders: [Order!]!
    }

    extend type Query {
        addresses: [Address!]!
    }

    extend type Mutation {
        addAddress(input: AddressInput!): Address!
        updateAddress(input: AddressInput!): Address!
        deleteAddresses(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        addresses: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addAddress: async (_, args, context, info) => {
            // Must be admin, or adding to your own
            if(!context.req.isAdmin && (context.req.businessId !== args.input.businessId)) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        updateAddress: async (_, args, context, info) => {
            // Must be admin, or updating your own
            const curr = await context.prisma[_model].findUnique({ where: { id: args.input.id } });
            if (!context.req.isAdmin && context.req.businessId !== curr.businessId) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteAddresses: async (_, args, context) => {
            // Must be admin, or deleting your own
            const specified = await context.prisma[_model].findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const business_ids = [...new Set(specified.map(s => s.businessId))];
            if (!context.req.isAdmin && (business_ids.length > 1 || context.req.business_id !== business_ids[0])) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
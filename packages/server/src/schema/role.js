import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { TABLES } from '../db';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Role;

export const typeDef = gql`
    input RoleInput {
        id: ID
        title: String!
        description: String
        customerIds: [ID!]
    }

    type CustomerRole {
        customer: Customer!
        role: Role!
    }

    type Role {
        id: ID!
        title: String!
        description: String
        customers: [Customer!]!
    }

    extend type Query {
        roles: [Role!]!
    }

    extend type Mutation {
        addRole(input: RoleInput!): Role!
        updateRole(input: RoleInput!): Role!
        deleteRoles(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        roles: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addRole: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        updateRole: async (_, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteRoles: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
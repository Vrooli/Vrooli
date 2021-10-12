import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

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
        roles: async (_parent, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.role.findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addRole: async (_parent, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.role.create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        updateRole: async (_parent, args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.role.update({
                where: { id: args.input.id || undefined },
                data: { ...args.input },
                ...(new PrismaSelect(info).value)
            })
        },
        deleteRoles: async (_parent, args, context, _info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.role.deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}
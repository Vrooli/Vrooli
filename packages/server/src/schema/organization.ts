import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input OrganizationInput {
        id: ID
        name: String!
        description: String
        resources: [ResourceInput!]
    }

    type Organization {
        id: ID!
        name: String!
        description: String
        resources: [Resource!]
        projects: [Project!]
        wallets: [Wallet!]
        starredBy: [User!]
        routines: [Routine!]
    }

    extend type Mutation {
        addOrganization(input: OrganizationInput!): Organization!
        updateOrganization(input: OrganizationInput!): Organization!
        deleteOrganization(id: ID!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addOrganization: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        updateOrganization: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        deleteOrganization: async (_parent: undefined, args: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}
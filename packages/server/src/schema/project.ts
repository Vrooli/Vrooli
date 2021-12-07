import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input ProjectInput {
        id: ID
        name: String!
        description: String
        organizations: [OrganizationInput!]
        users: [UserInput!]
        resources: [ResourceInput!]
    }

    type Project {
        id: ID!
        name: String!
        description: String
        resources: [Resource!]
        wallets: [Wallet!]
        users: [User!]
        organizations: [Organization!]
        starredBy: [User!]
    }

    extend type Mutation {
        addProject(input: ProjectInput!): Project!
        updateProject(input: ProjectInput!): Project!
        deleteProject(id: ID!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addProject: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        updateProject: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        deleteProject: async (_parent: undefined, args: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}
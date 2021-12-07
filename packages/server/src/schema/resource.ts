import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input ResourceInput {
        id: ID
        name: String!
        description: String
        link: String!
        displayUrl: String
        userId: ID
        organizationId: ID
    }

    type Resource {
        id: ID!
        name: String!
        description: String
        link: String!
        displayUrl: String
    }

    extend type Mutation {
        addResource(input: ResourceInput!): Resource!
        updateResource(input: ResourceInput!): Resource!
        deleteResources(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Mutation: {
        addResource: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        updateResource: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        deleteResources: async (_parent: undefined, args: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}
import { gql } from 'apollo-server-express';
import { CreateOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { ApiKey, ApiKeyCreateInput, ApiKeyDeleteOneInput, ApiKeyUpdateInput, ApiKeyValidateInput, Success } from '@shared/consts';
import { rateLimit } from '../middleware';
import { createHelper, deleteOneHelper, updateHelper } from '../actions';
import { assertRequestFrom } from '../auth';
import { COOKIE } from '@shared/consts';
import { CustomError } from '../events';

export const typeDef = gql`
    input ApiKeyCreateInput {
        creditsUsedBeforeLimit: Int!
        stopAtLimit: Boolean!
        absoluteMax: Int!
    }
    input ApiKeyUpdateInput {
        id: ID!
        creditsUsedBeforeLimit: Int
        stopAtLimit: Boolean
        absoluteMax: Int
    }
    input ApiKeyDeleteOneInput {
        id: ID!
    }
    input ApiKeyValidateInput {
        id: ID!
        secret: String!
    }

    type ApiKey {
        id: ID!
        creditsUsed: Int!
        creditsUsedBeforeLimit: Int!
        stopAtLimit: Boolean!
        absoluteMax: Int!
        resetsAt: Date!
    }

    extend type Mutation {
        apiKeyCreate(input: ApiKeyCreateInput!): ApiKey!
        apiKeyUpdate(input: ApiKeyUpdateInput!): ApiKey!
        apiKeyDeleteOne(input: ApiKeyDeleteOneInput!): Success!
        apiKeyValidate(input: ApiKeyValidateInput!): ApiKey!
    }
`

const objectType = 'ApiKey';
export const resolvers: {
    Mutation: {
        apiKeyCreate: GQLEndpoint<ApiKeyCreateInput, CreateOneResult<ApiKey>>;
        apiKeyUpdate: GQLEndpoint<ApiKeyUpdateInput, UpdateOneResult<ApiKey>>;
        apiKeyDeleteOne: GQLEndpoint<ApiKeyDeleteOneInput, Success>;
        apiKeyValidate: GQLEndpoint<ApiKeyValidateInput, ApiKey>;
    }
} = {
    Mutation: {
        apiKeyCreate: async (_, { input }, { prisma, req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ info, maxUser: 10, req });
            return createHelper({ info, input, objectType, prisma, req })
        },
        apiKeyUpdate: async (_, { input }, { prisma, req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ info, maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req })
        },
        apiKeyDeleteOne: async (_, { input }, { prisma, req }, info) => {
            assertRequestFrom(req, { isOfficialUser: true });
            await rateLimit({ info, maxUser: 10, req });
            return deleteOneHelper({ input, objectType, prisma, req });
        },
        apiKeyValidate: async (_, { input }, { req, res }, info) => {
            await rateLimit({ info, maxApi: 5000, req });
            // If session is expired
            if (!req.apiToken || !req.validToken) {
                res.clearCookie(COOKIE.Jwt);
                throw new CustomError('0318', 'SessionExpired', req.languages);
            }
            // TODO
            throw new CustomError('0319', 'NotImplemented', req.languages);
        },
    }
}
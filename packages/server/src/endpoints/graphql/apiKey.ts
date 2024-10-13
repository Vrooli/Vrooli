import { ApiKeyEndpoints, EndpointsApiKey } from "../logic/apiKey";

export const typeDef = `#graphql
    input ApiKeyCreateInput {
        absoluteMax: Int!
        creditsUsedBeforeLimit: Int!
        stopAtLimit: Boolean!
        teamConnect: ID
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
`;

export const resolvers: {
    Mutation: EndpointsApiKey["Mutation"];
} = {
    ...ApiKeyEndpoints,
};

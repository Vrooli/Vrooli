import { gql } from "apollo-server-express";
import { EndpointsPushDevice, PushDeviceEndpoints } from "../logic";

export const typeDef = gql`
    input PushDeviceKeysInput {
        auth: String!   
        p256dh: String!
    }
    input PushDeviceCreateInput {
        endpoint: String!
        expires: Date
        keys: PushDeviceKeysInput!
        name: String
    }
    input PushDeviceUpdateInput {
        id: ID!
        name: String
    }

    type PushDevice {
        id: ID!
        deviceId: String!
        expires: Date
        name: String
    }

    extend type Query {
        pushDevices: [PushDevice!]!
    }

    extend type Mutation {
        pushDeviceCreate(input: PushDeviceCreateInput!): PushDevice!
        pushDeviceUpdate(input: PushDeviceUpdateInput!): PushDevice!
    }
`;

export const resolvers: {
    Query: EndpointsPushDevice["Query"];
    Mutation: EndpointsPushDevice["Mutation"];
} = {
    ...PushDeviceEndpoints,
};

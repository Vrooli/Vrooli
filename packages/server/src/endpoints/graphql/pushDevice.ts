import { gql } from "apollo-server-express";
import { EndpointsPushDevice, PushDeviceEndpoints } from "../logic/pushDevice";

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
    input PushDeviceTestInput {
        id: ID!
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
        pushDeviceTest(input: PushDeviceTestInput!): Success!
        pushDeviceUpdate(input: PushDeviceUpdateInput!): PushDevice!
    }
`;

export const resolvers: {
    Query: EndpointsPushDevice["Query"];
    Mutation: EndpointsPushDevice["Mutation"];
} = {
    ...PushDeviceEndpoints,
};

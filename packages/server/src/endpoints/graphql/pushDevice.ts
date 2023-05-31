import { PushDeviceCreateInput, PushDeviceUpdateInput } from "@local/shared";
import { gql } from "apollo-server-express";
import { updateHelper } from "../../actions";
import { assertRequestFrom } from "../../auth";
import { rateLimit } from "../../middleware";
import { Notify } from "../../notify";
import { CreateOneResult, GQLEndpoint, RecursivePartial, UpdateOneResult } from "../../types";

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

const objectType = "PushDevice";
export const resolvers: {
    Query: {
        pushDevices: GQLEndpoint<never, RecursivePartial<any>>;
    }
    Mutation: {
        pushDeviceCreate: GQLEndpoint<PushDeviceCreateInput, CreateOneResult<any>>;
        pushDeviceUpdate: GQLEndpoint<PushDeviceUpdateInput, UpdateOneResult<any>>;
    }
} = {
    Query: {
        pushDevices: async (_p, _d, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            return prisma.push_device.findMany({
                where: { userId: id },
                select: { id: true, name: true, expires: true },
            });
        },
    },
    Mutation: {
        pushDeviceCreate: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            return Notify(prisma, userData.languages).registerPushDevice({
                endpoint: input.endpoint,
                expires: input.expires,
                auth: input.keys.auth,
                p256dh: input.keys.p256dh,
                // name: input.name,
                userData,
                info,
            });
        },
        pushDeviceUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

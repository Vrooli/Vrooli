import { PushDevice, PushDeviceCreateInput, PushDeviceUpdateInput } from "@local/shared";
import { updateHelper } from "../../actions/updates";
import { assertRequestFrom } from "../../auth/request";
import { rateLimit } from "../../middleware/rateLimit";
import { Notify } from "../../notify";
import { CreateOneResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsPushDevice = {
    Query: {
        pushDevices: GQLEndpoint<never, FindOneResult<PushDevice>[]>;
    }
    Mutation: {
        pushDeviceCreate: GQLEndpoint<PushDeviceCreateInput, CreateOneResult<PushDevice>>;
        pushDeviceUpdate: GQLEndpoint<PushDeviceUpdateInput, UpdateOneResult<PushDevice>>;
    }
}

const objectType = "PushDevice";
export const PushDeviceEndpoints: EndpointsPushDevice = {
    Query: {
        pushDevices: async (_p, _d, { prisma, req }) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            await rateLimit({ maxUser: 1000, req });
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
            await rateLimit({ maxUser: 10, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
    },
};

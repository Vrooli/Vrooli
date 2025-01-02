import { PushDevice, PushDeviceCreateInput, PushDeviceTestInput, PushDeviceUpdateInput, Success } from "@local/shared";
import { updateOneHelper } from "../../actions/updates";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Notify } from "../../notify";
import { ApiEndpoint, CreateOneResult, FindOneResult, UpdateOneResult } from "../../types";

export type EndpointsPushDevice = {
    findMany: ApiEndpoint<undefined, FindOneResult<PushDevice>[]>;
    createOne: ApiEndpoint<PushDeviceCreateInput, CreateOneResult<PushDevice>>;
    testOne: ApiEndpoint<PushDeviceTestInput, Success>;
    updateOne: ApiEndpoint<PushDeviceUpdateInput, UpdateOneResult<PushDevice>>;
}

const objectType = "PushDevice";
export const pushDevice: EndpointsPushDevice = {
    findMany: async (_p, _d, { req }) => {
        const { id } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return prismaInstance.push_device.findMany({
            where: { userId: id },
            select: { id: true, name: true, expires: true },
        });
    },
    createOne: async (_, { input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        return Notify(userData.languages).registerPushDevice({
            endpoint: input.endpoint,
            expires: input.expires,
            auth: input.keys.auth,
            p256dh: input.keys.p256dh,
            name: input.name ?? undefined,
            userData,
            info,
        });
    },
    testOne: async (_, { input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const pushDevices = await prismaInstance.push_device.findMany({
            where: { userId: userData.id },
        });
        const requestedId = input.id;
        const requestedDevice = pushDevices.find(({ id }) => id === requestedId);
        if (!requestedDevice) {
            logger.info(`devices: ${requestedId}, ${JSON.stringify(pushDevices)}`);
            throw new CustomError("0588", "Unauthorized");
        }
        const success = await Notify(userData.languages).testPushDevice(requestedDevice);
        return success;
    },
    updateOne: async (_, { input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};

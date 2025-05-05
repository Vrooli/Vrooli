import { PushDevice, PushDeviceCreateInput, PushDeviceTestInput, PushDeviceUpdateInput, Success } from "@local/shared";
import { updateOneHelper } from "../../actions/updates.js";
import { RequestService } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ApiEndpoint } from "../../types.js";

export type EndpointsPushDevice = {
    findMany: ApiEndpoint<undefined, PushDevice[]>;
    createOne: ApiEndpoint<PushDeviceCreateInput, PushDevice>;
    testOne: ApiEndpoint<PushDeviceTestInput, Success>;
    updateOne: ApiEndpoint<PushDeviceUpdateInput, PushDevice>;
}

const objectType = "PushDevice";
export const pushDevice: EndpointsPushDevice = {
    findMany: async (_d, { req }) => {
        const { id } = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        const devices = await DbProvider.get().push_device.findMany({
            where: { userId: BigInt(id) },
            select: { id: true, name: true, expires: true },
        });
        return devices.map(({ id, name, expires }) => ({ id: id.toString(), name, expires }));
    },
    createOne: async ({ input }, { req }, info) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const { Notify } = await import("../../notify/notify.js");
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
    testOne: async ({ input }, { req }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        const pushDevices = await DbProvider.get().push_device.findMany({
            where: { userId: BigInt(userData.id) },
        });
        const requestedId = input.id;
        const requestedDevice = pushDevices.find(({ id }) => id.toString() === requestedId);
        if (!requestedDevice) {
            logger.info(`devices: ${requestedId}, ${JSON.stringify(pushDevices)}`);
            throw new CustomError("0588", "Unauthorized");
        }
        const { Notify } = await import("../../notify/notify.js");
        const success = await Notify(userData.languages).testPushDevice(requestedDevice);
        return success;
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 10, req });
        return updateOneHelper({ info, input, objectType, req });
    },
};

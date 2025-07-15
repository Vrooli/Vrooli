import { type PushDevice, type PushDeviceCreateInput, type PushDeviceTestInput, type PushDeviceUpdateInput, type Success } from "@vrooli/shared";
import { RequestService } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { type ApiEndpoint } from "../../types.js";
import { createStandardCrudEndpoints, PermissionPresets } from "../helpers/endpointFactory.js";

// AI_CHECK: pushdevice_endpoint_types=1 | LAST: 2025-06-29 - Fixed findMany return type to match PushDevice interface

export type EndpointsPushDevice = {
    findMany: ApiEndpoint<undefined, PushDevice[]>;
    createOne: ApiEndpoint<PushDeviceCreateInput, PushDevice>;
    testOne: ApiEndpoint<PushDeviceTestInput, Success>;
    updateOne: ApiEndpoint<PushDeviceUpdateInput, PushDevice>;
}

const objectType = "PushDevice";
export const pushDevice: EndpointsPushDevice = createStandardCrudEndpoints({
    objectType,
    endpoints: {
        updateOne: {
            rateLimit: { maxUser: 10 },
            permissions: PermissionPresets.WRITE_PRIVATE,
        },
    },
    customEndpoints: {
        findMany: async (_d, { req }) => {
            const { id } = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            const devices = await DbProvider.get().push_device.findMany({
                where: { userId: BigInt(id) },
                select: { id: true, name: true, expires: true, endpoint: true },
            });
            return devices.map(({ id, name, expires, endpoint }) => ({
                __typename: "PushDevice" as const,
                id: id.toString(),
                deviceId: endpoint, // endpoint is the deviceId
                name,
                expires: expires?.toISOString() ?? null,
            }));
        },
        createOne: async (data, { req }, info) => {
            const input = data?.input;
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
        testOne: async (data, { req }) => {
            const input = data?.input;
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
            const success = await Notify(userData.languages).testPushDevice(requestedDevice.id.toString(), userData.id);
            return success;
        },
    },
});

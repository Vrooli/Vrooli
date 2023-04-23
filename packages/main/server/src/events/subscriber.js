import { getLogic } from "../getters";
import { subscribableMapper } from "../models";
import { CustomError } from "./error";
export const Subscriber = (prisma) => ({
    subscribe: async (object, userData, silent) => {
        const { delegate, validate } = getLogic(["delegate", "validate"], object.__typename, userData.languages, "Transfer.request-object");
        const permissionData = await delegate(prisma).findUnique({
            where: { id: object.id },
            select: validate.permissionsSelect,
        });
        const isPublic = permissionData && validate.isPublic(permissionData, userData.languages);
        const isDeleted = permissionData && validate.isDeleted(permissionData, userData.languages);
        if (!isPublic || isDeleted)
            throw new CustomError("0332", "Unauthorized", userData.languages);
        await prisma.notification_subscription.create({
            data: {
                subscriber: { connect: { id: userData.id } },
                [subscribableMapper[object.__typename]]: { connect: { id: object.id } },
                silent,
            },
        });
    },
    unsubscribe: async (subscriptionId, userData) => {
        const subscription = await prisma.notification_subscription.findUnique({
            where: { id: subscriptionId },
            select: {
                subscriberId: true,
            },
        });
        if (!subscription)
            throw new CustomError("0333", "NotFound", userData.languages);
        if (subscription.subscriberId !== userData.id)
            throw new CustomError("0334", "Unauthorized", userData.languages);
        await prisma.notification_subscription.delete({
            where: { id: subscriptionId },
        });
    },
    update: async (subscriptionId, userData, silent) => {
        const subscription = await prisma.notification_subscription.findUnique({
            where: { id: subscriptionId },
            select: {
                subscriberId: true,
                silent: true,
            },
        });
        if (!subscription)
            throw new CustomError("0335", "NotFound", userData.languages);
        if (subscription.subscriberId !== userData.id)
            throw new CustomError("0336", "Unauthorized", userData.languages);
        if (subscription.silent !== silent) {
            await prisma.notification_subscription.update({
                where: { id: subscriptionId },
                data: { silent },
            });
        }
    },
});
//# sourceMappingURL=subscriber.js.map
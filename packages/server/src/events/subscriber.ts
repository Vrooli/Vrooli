import { SubscribableObject } from "@local/shared";
import { getLogic } from "../getters";
import { subscribableMapper } from "../models";
import { PrismaType, SessionUserToken } from "../types";
import { CustomError } from "./error";

/**
 * Handles notifying users of new activity on object they're subscribed to. 
 * For each object, a user should receive a notification when:
 * - A new version (which is public and complete) is created, or an existing version which was public or incomplete is made public and complete
 * - A new comment is created (shouldn't send a notification every time - have to come up with clever way to do this)
 * - A new report is created
 * - A new pull request is created
 * - A new issue is created
 * - A new question is created
 * - A new answer is created
 */
export const Subscriber = (prisma: PrismaType) => ({
    /**
     * Creates a subscription for a user to an object
     * @param object The object to subscribe to
     * @param userData The user subscribing to the object
     * @param silent True if push notifications should not be sent
     */
    subscribe: async (
        object: { __typename: SubscribableObject, id: string },
        userData: SessionUserToken,
        silent?: boolean,
    ) => {
        // Find the object and its owner
        const { delegate, validate } = getLogic(["delegate", "validate"], object.__typename, userData.languages, "Transfer.request-object");
        const permissionData = await delegate(prisma).findUnique({
            where: { id: object.id },
            select: validate.permissionsSelect,
        });
        const isPublic = permissionData && validate.isPublic(permissionData, userData.languages);
        const isDeleted = permissionData && validate.isDeleted(permissionData, userData.languages);
        // Don't subscribe if object is private or deleted
        if (!isPublic || isDeleted)
            throw new CustomError("0332", "Unauthorized", userData.languages);
        // Create subscription
        await prisma.notification_subscription.create({
            data: {
                subscriber: { connect: { id: userData.id } },
                [subscribableMapper[object.__typename]]: { connect: { id: object.id } },
                silent,
            },
        });
    },
    /**
     * Cancels a subscription for a user to an object
     * @param subscriptionId The ID of the subscription to cancel
     * @param userData The user unsubscribing from the object
     */
    unsubscribe: async (
        subscriptionId: string,
        userData: SessionUserToken,
    ) => {
        // Find the subscription
        const subscription = await prisma.notification_subscription.findUnique({
            where: { id: subscriptionId },
            select: {
                subscriberId: true,
            },
        });
        // Make sure the subscription exists and is owned by the user
        if (!subscription)
            throw new CustomError("0333", "NotFound", userData.languages);
        if (subscription.subscriberId !== userData.id)
            throw new CustomError("0334", "Unauthorized", userData.languages);
        // Delete subscription
        await prisma.notification_subscription.delete({
            where: { id: subscriptionId },
        });
    },
    /**
     * Changes silent status of a subscription
     * @param subscriptionId The ID of the subscription to change
     * @param userData The user changing the subscription
     * @param silent True if push notifications should not be sent
     */
    update: async (
        subscriptionId: string,
        userData: SessionUserToken,
        silent: boolean,
    ) => {
        // Find the subscription
        const subscription = await prisma.notification_subscription.findUnique({
            where: { id: subscriptionId },
            select: {
                subscriberId: true,
                silent: true,
            },
        });
        // Make sure the subscription exists and is owned by the user
        if (!subscription)
            throw new CustomError("0335", "NotFound", userData.languages);
        if (subscription.subscriberId !== userData.id)
            throw new CustomError("0336", "Unauthorized", userData.languages);
        // Update subscription if silent status has changed
        if (subscription.silent !== silent) {
            await prisma.notification_subscription.update({
                where: { id: subscriptionId },
                data: { silent },
            });
        }
    },
});

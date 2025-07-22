import type { Prisma } from "@prisma/client";
import { generatePK, type SessionUser, type SubscribableObject } from "@vrooli/shared";
import { permissionsSelectHelper } from "../builders/permissionsSelectHelper.js";
import { type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { ModelMap } from "../models/base/index.js";
import { CustomError } from "./error.js";

export const subscribableMapper: { [key in SubscribableObject]: keyof Prisma.notification_subscriptionUpsertArgs["create"] } = {
    Comment: "comment",
    Issue: "issue",
    Meeting: "meeting",
    PullRequest: "pullRequest",
    Report: "report",
    Resource: "resource",
    Schedule: "schedule",
    Team: "team",
};

/**
 * Handles notifying users of new activity on object they're subscribed to. 
 * For each object, a user should receive a notification when:
 * - A new version (which is public and complete) is created, or an existing version which was public or incomplete is made public and complete
 * - A new comment is created (shouldn't send a notification every time - have to come up with clever way to do this)
 * - A new report is created
 * - A new pull request is created
 * - A new issue is created
 * - A new answer is created
 */
export function Subscriber() {
    return {
        /**
         * Creates a subscription for a user to an object
         * @param object The object to subscribe to
         * @param userData The user subscribing to the object
         * @param silent True if push notifications should not be sent
         */
        subscribe: async (
            object: { __typename: SubscribableObject, id: string },
            userData: SessionUser,
            silent?: boolean,
        ) => {
            // Find the object and its owner
            const { dbTable, validate } = ModelMap.getLogic(["dbTable", "validate"], object.__typename);
            const permissionData = await (DbProvider.get()[dbTable] as PrismaDelegate).findUnique({
                where: { id: object.id },
                select: permissionsSelectHelper(validate().permissionsSelect, userData.id),
            });
            const isPublic = permissionData && validate().isPublic(permissionData, () => undefined);
            const isDeleted = permissionData && validate().isDeleted(permissionData);
            // Don't subscribe if object is private or deleted
            if (!isPublic || isDeleted)
                throw new CustomError("0332", "Unauthorized");
            // Create subscription
            await DbProvider.get().notification_subscription.create({
                data: {
                    id: generatePK(),
                    subscriber: { connect: { id: BigInt(userData.id) } },
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
            userData: SessionUser,
        ) => {
            // Find the subscription
            const subscription = await DbProvider.get().notification_subscription.findUnique({
                where: { id: BigInt(subscriptionId) },
                select: {
                    subscriberId: true,
                },
            });
            // Make sure the subscription exists and is owned by the user
            if (!subscription)
                throw new CustomError("0333", "NotFound");
            if (subscription.subscriberId.toString() !== userData.id)
                throw new CustomError("0334", "Unauthorized");
            // Delete subscription
            await DbProvider.get().notification_subscription.delete({
                where: { id: BigInt(subscriptionId) },
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
            userData: SessionUser,
            silent: boolean,
        ) => {
            // Find the subscription
            const subscription = await DbProvider.get().notification_subscription.findUnique({
                where: { id: BigInt(subscriptionId) },
                select: {
                    subscriberId: true,
                    silent: true,
                },
            });
            // Make sure the subscription exists and is owned by the user
            if (!subscription)
                throw new CustomError("0335", "NotFound");
            if (subscription.subscriberId.toString() !== userData.id)
                throw new CustomError("0336", "Unauthorized");
            // Update subscription if silent status has changed
            if (subscription.silent !== silent) {
                await DbProvider.get().notification_subscription.update({
                    where: { id: BigInt(subscriptionId) },
                    data: { silent },
                });
            }
        },
    };
}


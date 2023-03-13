import { GqlModelType, SessionUser } from "@shared/consts";
import { Trigger } from "../events";
import { TransferModel } from "../models";
import { PrismaType } from "../types";

/**
 * Handles trigger and transfer creation for root objects
 */
export const onRootCreated = async ({ created, objectType, preMap, prisma, userData }: {
    created: { [x: string]: any }[],
    objectType: GqlModelType | `${GqlModelType}`,
    preMap: { [x in `${GqlModelType}`]?: any },
    prisma: PrismaType,
    userData: SessionUser,
}) => {
    // Loop through created items
    for (let i = 0; i < created.length; i++) {
        const objectId = created[i].id;
        // Get trigger info
        const {
            hasCompleteAndPublic,
            hasParent,
            owner,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectCreated
        await Trigger(prisma, userData.languages).objectCreated({
            createdById: userData.id,
            hasCompleteAndPublic,
            hasParent,
            owner,
            objectId,
            objectType,
            // Projects are attached to versions, not root objects
            projectId: undefined,
        });
        // Get transfer info 
        const requiresTransfer = preMap[objectType].transferMap[objectId];
        if (requiresTransfer) {
            // Create transfer
            await TransferModel.transfer(prisma).requestSend({
                objectConnect: objectId,
                objectType,
                toOrganizationConnect: owner.__typename === 'Organization' ? owner.id : undefined,
                toUserConnect: owner.__typename === 'User' ? owner.id : undefined,
            }, userData);
        }
    }
}
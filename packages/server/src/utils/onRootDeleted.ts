import { GqlModelType, SessionUser } from "@shared/consts";
import { Trigger } from "../events";
import { PrismaType } from "../types";

/**
 * Handles trigger and transfer creation for root objects
 */
export const onRootDeleted = async ({ deletedIds, objectType, preMap, prisma, userData }: {
    deletedIds: string[],
    objectType: GqlModelType | `${GqlModelType}`,
    preMap: { [x in `${GqlModelType}`]?: any },
    prisma: PrismaType,
    userData: SessionUser,
}) => {
    // Loop through deleted items
    for (let i = 0; i < deletedIds.length; i++) {
        const objectId = deletedIds[i];
        // Get trigger info
        const {
            hasBeenTransferred,
            hasParent,
            wasCompleteAndPublic,
        } = preMap[objectType].triggerMap[objectId];
        // Trigger objectDeleted
        await Trigger(prisma, userData.languages).objectDeleted({
            deletedById: userData.id,
            hasBeenTransferred,
            hasParent,
            objectId,
            objectType,
            wasCompleteAndPublic,
        });
    }
},
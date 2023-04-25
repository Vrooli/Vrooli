
import { GqlModelType } from "@local/shared";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { IdsByAction, IdsByType } from "./types";

/**
 * Helper function to convert placeholder ids to actual ids. Required because certain Prisma operations 
 * on single-object relations (disconnect, delete, update) don't explicity specify the id of the relationship object
 * @returns Maps with placeholders replaced with ids
 */
export const convertPlaceholders = async ({
    idsByAction,
    idsByType,
    prisma,
    languages
}: {
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    prisma: PrismaType,
    languages: string[]
}): Promise<{
    idsByAction: IdsByAction,
    idsByType: IdsByType,
}> => {
    const updatedIdsByAction: IdsByAction = { ...idsByAction };
    const updatedIdsByType: IdsByType = { ...idsByType };

    // Iterate over the idsByAction map
    for (const actionType in idsByAction) {
        const ids = idsByAction[actionType];
        const updatedIds: string[] = [];

        // Iterate over the IDs in the current actionType
        for (const id of ids) {
            // Check if the ID is a placeholder
            if (typeof id === "string" && id.includes("|")) {
                // Split the placeholder string into the object type and path
                const [objectType, path] = id.split("|", 2);
                // Split the path into rootId and relationship steps
                const [rootId, ...relations] = path.split(".");

                // Query the database to find the actual ID
                const { delegate } = getLogic(['delegate'], objectType as GqlModelType, languages, 'convertPlaceholders');
                const queryResult = await delegate(prisma).findUnique({
                    where: { id: rootId },
                    select: relations.reduce((selectObj, relation) => {
                        return { [relation]: selectObj };
                    }, {}),
                });

                // Traverse the result object following the relationship steps
                let currentObject = queryResult;
                for (const relation of relations) {
                    currentObject = currentObject![relation];
                }

                // Add the actual ID to the updatedIds array if it exists
                if (currentObject && currentObject.id) {
                    updatedIds.push(currentObject.id);
                }
            } else {
                // If not a placeholder, add the ID directly to the updatedIds array
                updatedIds.push(id);
            }
        }

        // Update the actionType entry with the updated IDs
        updatedIdsByAction[actionType] = updatedIds;
    }

    // Iterate over the idsByType map
    for (const objectType in idsByType) {
        const ids = idsByType[objectType];
        const updatedIds: string[] = [];

        // Iterate over the IDs in the current objectType
        for (const id of ids) {
            // Check if the ID is a placeholder
            if (typeof id === "string" && id.includes("|")) {
                // Split the placeholder string into rootId and relationship steps
                const [rootId, ...relations] = id.split("|")[1].split(".");

                // Query the database to find the actual ID
                const queryResult = await prisma[objectType].findUnique({
                    where: { id: rootId },
                    select: relations.reduce((selectObj, relation) => {
                        return { [relation]: selectObj };
                    }, {}),
                });

                // Traverse the result object following the relationship steps
                let currentObject = queryResult;
                for (const relation of relations) {
                    currentObject = currentObject[relation];
                }

                // Add the actual ID to the updatedIds array if it exists
                if (currentObject && currentObject.id) {
                    updatedIds.push(currentObject.id);
                }
            } else {
                updatedIds.push(id);
            }
        }

        updatedIdsByType[objectType] = updatedIds;
    }

    return {
        idsByAction: updatedIdsByAction,
        idsByType: updatedIdsByType,
    };
};
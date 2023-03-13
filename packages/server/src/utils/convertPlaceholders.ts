
import { GqlModelType } from "@shared/consts";
import { PrismaType } from "../types";
import { IdsByAction, IdsByType } from "./types";

// TODO need placeholder for isOneToOne updating AND disconnecting AND deleting. isOneToOne might also not be going through validation for create/update (max checks, etc.). Current implementation should be okay for connects
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
    // Initialize object to hold prisma queries
    const queries: { [key in GqlModelType]?: { ids: string[], select: { [x: string]: any } } } = {};
    // Loop through all keys in ids
    Object.keys(idActions).forEach(key => {
        // Loop through all ids in key
        idActions[key as any].forEach((idAction: string) => {
            // If id is a placeholder, add to placeholderIds
            if (idAction.includes('.')) {
                queries[key as any] = queries[key as any] ?? { ids: [], select: {} };
                // Get id, which is everything after the first '-' and before '.'
                const id = idAction.split('-')[1].split('.')[0];
                queries[key as any].ids.push(id);
                // Get placeholder, which is everything after '.'
                const placeholder = idAction.split('.')[1];
                // Add placeholder to select object
                queries[key as any].select[id] = { [placeholder]: true };
            }
        });
    });
    // If there are no placeholders, return ids to save time
    if (Object.keys(queries).length === 0) return idActions;
    // Initialize object to hold query results
    const queryData: { [key in GqlModelType]?: { [x: string]: any } } = {};
    // Loop through all keys in queries
    for (const key in Object.keys(queries)) {
        // If there are any no ids, skip
        if (queries[key as any].ids.length === 0) continue;
        // Query for ids
        const { delegate } = getLogic(['delegate'], key as GqlModelType, languages, 'disconnectPlaceholdersToIds');
        queryData[key as any] = await delegate(prisma).findMany({
            where: { id: { in: queries[key as any].ids } },
            select: queries[key as any].select,
        });
    }
    // Initialize return object
    const result: { [key in GqlModelType]?: string[] } = {};
    // Loop through all keys in idActions
    Object.keys(idActions).forEach(key => {
        // Get queryData for this key
        const data: { [x: string]: any }[] | undefined = queryData[key as any];
        // Loop through all ids in key
        idActions[key as any].forEach((idAction: string) => {
            // If id is a placeholder, replace with id from results
            if (data && idAction.includes('.')) {
                // Get id, which is everything after the first '-' and before '.'
                const id = idAction.split('-')[1].split('.')[0];
                // Get placeholder, which is everything after '.'
                const placeholder = idAction.split('.')[1];
                // Find data object with matching id, and use placeholder to get disconnect id
                const parent = data.find(d => d.id === id) as { [x: string]: any };
                const disconnectId = parent[placeholder];
                // Add disconnect id to result
                result[key as any] = result[key as any] ?? [];
                result[key as any].push(`Disconnect-${disconnectId}`);
            }
            // Otherwise, add to result
            else { result[key as any] = result[key as any] ?? []; result[key as any].push(idAction); }
        });
    });
    return result;
};

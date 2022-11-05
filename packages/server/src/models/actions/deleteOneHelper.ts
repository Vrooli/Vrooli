import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, logger, LogLevel } from "../../events";
import { Success, LogType } from "../../schema/types";
import { getUserId } from "../builder";
import { DeleteOneHelperProps } from "./types";

/**
 * Helper function for deleting one object in a single line
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper({
    input,
    model,
    prisma,
    req,
}: DeleteOneHelperProps): Promise<Success> {
    const userId = getUserId(req);
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete object', { code: genErrorCode('0033') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0034') });
    // Delete object. cud will check permissions
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userId, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = model.format.relationshipMap.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            // No need to await this, since it is not needed for the response
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Delete,
                object1Type: objectType,
                object1Id: input.id,
            }).catch(error => logger.log(LogLevel.error, 'Failed creating "Delete" log', { code: genErrorCode('0196'), error }));
        }
        return { success: true }
    }
    return { success: false };
}
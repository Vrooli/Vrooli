import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, logger, LogLevel } from "../../events";
import { Count, LogType } from "../../schema/types";
import { getUserId } from "../builder";
import { DeleteManyHelperProps } from "./types";

/**
 * Helper function for deleting many of the same object in a single line
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper({
    input,
    model,
    prisma,
    req,
}: DeleteManyHelperProps): Promise<Count> {
    const userId = getUserId(req);
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete objects', { code: genErrorCode('0035') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0036') });
    // Delete objects. cud will check permissions
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userId, deleteMany: input.ids });
    if (!deleted)
        throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in deleteManyHelper', { code: genErrorCode('0037') });
    // If organization, project, routine, or standard, log for stats
    const objectType = model.format.relationshipMap.__typename;
    if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
        const logs = input.ids.map((id: string) => ({
            timestamp: Date.now(),
            userId: userId,
            action: LogType.Delete,
            object1Type: objectType,
            object1Id: id,
        }));
        // No need to await this, since it is not needed for the response
        Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Delete" logs', { code: genErrorCode('0197'), error }));
    }
    return deleted
}
import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { Success } from "../../schema/types";
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
        // Handle trigger
        await Trigger(prisma).objectDelete(input.objectType, input.id, userId);
        return { success: true }
    }
    return { success: false };
}
import { CODE, DeleteOneType } from "@shared/consts";
import { assertRequestFrom } from "../../auth/auth";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { Count } from "../../schema/types";
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
    const userData = assertRequestFrom(req, { isUser: true });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0036') });
    // Delete objects. cud will check permissions
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userData, deleteMany: input.ids });
    if (!deleted)
        throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in deleteManyHelper', { code: genErrorCode('0037') });
    const objectType = model.format.relationshipMap.__typename;
    // Handle trigger
    for (const id of input.ids) {
        await Trigger(prisma).objectDelete(objectType as DeleteOneType, id, userData.id);
    }
    return deleted
}
import { CODE } from "@shared/consts";
import { assertRequestFrom } from "../../auth/auth";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { Success } from "../../schema/types";
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
    const userData = assertRequestFrom(req, { isUser: true });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0034') });
    // Delete object. cud will check permissions
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userData, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // Handle trigger
        await Trigger(prisma).objectDelete(input.objectType, input.id, userData.id);
        return { success: true }
    }
    return { success: false };
}
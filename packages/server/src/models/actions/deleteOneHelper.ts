import { assertRequestFrom } from "../../auth/auth";
import { CustomError, Trigger } from "../../events";
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
        throw new CustomError('0035', 'DeleteNotSupported', userData.languages);
    // Delete object. cud will check permissions
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userData, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // Handle trigger
        await Trigger(prisma, req.languages).objectDelete(input.objectType, input.id, userData.id);
        return { success: true }
    }
    return { success: false };
}
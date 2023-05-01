import { Count } from "@local/shared";
import { assertRequestFrom } from "../auth/request";
import { CustomError } from "../events";
import { cudHelper } from "./cudHelper";
import { DeleteManyHelperProps } from "./types";

/**
 * Helper function for deleting many of the same object in a single line
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper({
    input,
    objectType,
    prisma,
    req,
}: DeleteManyHelperProps): Promise<Count> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Delete objects. cudHelper will check permissions and handle triggers
    const { deleted } = await cudHelper({ deleteMany: input.ids, objectType, partialInfo: {}, prisma, userData });
    if (!deleted)
        throw new CustomError("0037", "InternalError", userData.languages);
    return deleted;
}

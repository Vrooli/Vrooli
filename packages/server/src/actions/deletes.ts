import { Count, Success } from "@local/shared";
import { assertRequestFrom } from "../auth/request";
import { CustomError } from "../events";
import { cudHelper } from "./cuds";
import { DeleteManyHelperProps, DeleteOneHelperProps } from "./types";

/**
 * Helper function for deleting one object in a single line
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper({
    input,
    objectType,
    prisma,
    req,
}: DeleteOneHelperProps): Promise<Success> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Delete object. cudHelper will check permissions and handle triggers
    const { deleted } = await cudHelper({ deleteMany: [input.id], objectType, partialInfo: {}, prisma, userData });
    return { __typename: "Success" as const, success: Boolean(deleted?.count && deleted.count > 0) };
}


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

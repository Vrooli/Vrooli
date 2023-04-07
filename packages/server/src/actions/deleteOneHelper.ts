import { Success } from '@shared/consts';
import { assertRequestFrom } from "../auth/request";
import { cudHelper } from "./cudHelper";
import { DeleteOneHelperProps } from "./types";

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
    return { __typename: 'Success' as const, success: Boolean(deleted?.count && deleted.count > 0) };
}
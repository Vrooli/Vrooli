import { Count, Success } from "@local/shared";
import { assertRequestFrom } from "../auth/request";
import { cudHelper } from "./cuds";
import { DeleteManyHelperProps, DeleteOneHelperProps } from "./types";

/**
 * Helper function for deleting one object in a single line
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper({
    input,
    prisma,
    req,
}: DeleteOneHelperProps): Promise<Success> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Delete object. cudHelper will check permissions and handle triggers
    const deleted = (await cudHelper({
        inputData: [{
            actionType: "Delete",
            input: input.id,
            objectType: input.objectType,
        }],
        partialInfo: {},
        prisma,
        userData,
    }))[0];
    return { __typename: "Success" as const, success: deleted === true };
}


/**
 * Helper function for deleting many of the same object in a single line
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper({
    input,
    prisma,
    req,
}: DeleteManyHelperProps): Promise<Count> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Delete objects. cudHelper will check permissions and handle triggers
    const deleted = await cudHelper({
        // deleteMany: input.ids,
        inputData: input.objects.map(({ id, objectType }) => ({
            actionType: "Delete",
            input: id,
            objectType,
        })),
        partialInfo: {},
        prisma,
        userData,
    });
    return { __typename: "Count" as const, count: deleted.filter(d => d === true).length };
}

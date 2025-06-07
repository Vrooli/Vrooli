import { type Count, type SessionUser, type Success } from "@vrooli/shared";
import { SessionService } from "../auth/session.js";
import { cudHelper } from "./cuds.js";
import { type DeleteManyHelperProps, type DeleteOneHelperProps } from "./types.js";

/**
 * Helper function for deleting one object in a single line
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper({
    input,
    req,
}: DeleteOneHelperProps): Promise<Success> {
    const userData = SessionService.getUser(req) as SessionUser;
    // Delete object. cudHelper will check permissions and handle triggers
    const deleted = (await cudHelper({
        info: {},
        inputData: [{
            action: "Delete",
            input: input.id,
            objectType: input.objectType,
        }],
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
    req,
}: DeleteManyHelperProps): Promise<Count> {
    const userData = SessionService.getUser(req) as SessionUser;
    // Delete objects. cudHelper will check permissions and handle triggers
    const deleted = await cudHelper({
        // deleteMany: input.ids,
        info: {},
        inputData: input.objects.map(({ id, objectType }) => ({
            action: "Delete",
            input: id,
            objectType,
        })),
        userData,
    });
    return { __typename: "Count" as const, count: deleted.filter(d => d === true).length };
}

import { assertRequestFrom } from "../auth/request";
import { CustomError, Trigger } from "../events";
import { getDelegate, getValidator } from "../getters";
import { getAuthenticatedData } from "../utils";
import { maxObjectsCheck, permissionsCheck } from "../validators";
import { readOneHelper } from "./readOneHelper";
import { ForkHelperProps } from "./types";

/**
 * Helper function for forking an object in a single line
 * @returns GraphQL Success response object
 */
export async function forkHelper({
    info,
    input,
    objectType,
    prisma,
    req,
}: ForkHelperProps): Promise<any> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Find validator and prisma delegate for this object type. We will use these to 
    // find the IDs which need to be authenticated. Other functions use getAuthenticatedIds, 
    // but that function only looks at data being passed in, not data in the database.
    const validator = getValidator(objectType, req.languages, 'forkHelper');
    const prismaDelegate = getDelegate(objectType, prisma, req.languages, 'forkHelper');
    const duplicator = getDuplicator(objectType, prisma, req.languages, 'forkHelper');
    // Query for data required to validate ownership
    asdfasdf
    // Find owners of all objects being validated. Any object which is private 
    // must be owned either by the user making the request, or by the forked object's owner
    fdasfsafd
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.type]: [input.id] }, prisma, userData.id);
    // Check permissions
    permissionsCheck(authDataById, { ['Create']: [input.id] }, userData);
    // Additional check for paywall
    //TODO
    // Check max objects
    await maxObjectsCheck(authDataById, { ['Create']: [input.id] }, prisma, userData);
    const { object } = await model.mutate(prisma).duplicate!({ userId: userData.id, objectId: input.id, isFork: false, createCount: 0 });
    // Handle trigger
    await Trigger(prisma, req.languages).objectFork(input.objectType, input.id, userData.id);
    // Query for object
    const fullObject = await readOneHelper({
        info,
        input: { id: object.id },
        model,
        prisma,
        req,
    })
    return fullObject;
}
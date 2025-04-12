import { CopyHelperProps } from "./types.js";

/**
 * Helper function for forking an object in a single line
 * @returns API endpoint Success response object
 */
export async function copyHelper({
    info,
    input,
    objectType,
    req,
}: CopyHelperProps): Promise<any> {
    return;
    // const userData = SessionService.getUser(req) as SessionUser;
    // // Find validator and prisma delegate for this object type. We will use these to 
    // // find the IDs which need to be authenticated. Other functions use getAuthenticatedIds, 
    // // but that function only looks at data being passed in, not data in the database.
    // const validator = getValidator(objectType, req.session.languages, 'copyHelper');
    // const prismaDelegate = getDelegator(objectType, req.session.languages, 'copyHelper');
    // const duplicator = getDuplicator(objectType, req.session.languages, 'copyHelper');
    // // Query for data required to validate ownership
    // asdfasdf
    // // Find owners of all objects being validated. Any object which is private 
    // // must be owned either by the user making the request, or by the forked object's owner
    // fdasfsafd
    // // Query for all authentication data
    // const authDataById = await getAuthenticatedData({ [model.__typename]: [input.id] }, userData.id);
    // // Check permissions
    // await permissionsCheck(authDataById, { ['Create']: [input.id] }, userData);
    // // Additional check for paywall
    // //TODO
    // // Check max objects
    // await maxObjectsCheck(authDataById, { ['Create']: [input.id] }, userData);
    // const { object } = await model.mutate().duplicate!({ userId: userData.id, objectId: input.id, isFork: false, createCount: 0 });
    // // Handle trigger
    // await Trigger(req.session.languages).objectCopy(input.objectType, input.id, userData.id);
    // // Query for object
    // const fullObject = await readOneHelper({
    //     info,
    //     input: { id: object.id },
    //     model,
    //     req,
    // })
    // return fullObject;
}

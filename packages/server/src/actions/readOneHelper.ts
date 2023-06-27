import { ViewFor } from "@local/shared";
import { getUser } from "../auth";
import { addSupplementalFields, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getIdFromHandle, getLatestVersion } from "../getters";
import { ObjectMap, ViewModel } from "../models/base";
import { RecursivePartial } from "../types";
import { getAuthenticatedData } from "../utils";
import { permissionsCheck } from "../validators";
import { ReadOneHelperProps } from "./types";

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel extends { [x: string]: any }>({
    info,
    input,
    objectType,
    prisma,
    req,
}: ReadOneHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    const userData = getUser(req.session);
    const model = ObjectMap[objectType];
    if (!model) throw new CustomError("0350", "InternalError", req.session.languages, { objectType });
    // Validate input. This can be of the form FindByIdInput, FindByIdOrHandleInput, or FindVersionInput
    // Between these, the possible fields are id, idRoot, handle, and handleRoot
    if (!input.id && !input.idRoot && !input.handle && !input.handleRoot)
        throw new CustomError("0019", "IdOrHandleRequired", userData?.languages ?? req.session.languages);
    // Partially convert info
    const partialInfo = toPartialGqlInfo(info, model.format.gqlRelMap, req.session.languages, true);
    // If using idRoot or handleRoot, this means we are requesting a versioned object using data from the root object.
    // To query the version, we must find the latest completed version associated with the root object.
    let id: string | null | undefined;
    if (input.idRoot || input.handleRoot) {
        id = await getLatestVersion({ objectType: objectType as any, prisma, idRoot: input.idRoot, handleRoot: input.handleRoot });
    }
    // If using handle, find the id of the object with that handle
    else if (input.handle) {
        id = await getIdFromHandle({ handle: input.handle, objectType, prisma });
    }
    // Otherwise, use the id provided
    else {
        id = input.id;
    }
    if (!id)
        throw new CustomError("0434", "NotFound", userData?.languages ?? req.session.languages, { objectType });
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.__typename]: [id] }, prisma, userData ?? null);
    // Check permissions
    await permissionsCheck(authDataById, { ["Read"]: [id as string] }, userData);
    // Get the Prisma object
    let object: any;
    try {
        object = await model.delegate(prisma).findUnique({ where: { id }, ...selectHelper(partialInfo) });
        if (!object)
            throw new CustomError("0022", "NotFound", userData?.languages ?? req.session.languages, { objectType });
    } catch (error) {
        throw new CustomError("0435", "NotFound", userData?.languages ?? req.session.languages, { objectType, error });
    }
    // Return formatted for GraphQL
    const formatted = modelToGql(object, partialInfo) as RecursivePartial<GraphQLModel>;
    // If logged in and object tracks view counts, add a view
    if (userData?.id && objectType in ViewFor) {
        ViewModel.view(prisma, userData, { forId: object.id, viewFor: objectType as any });
    }
    const result = (await addSupplementalFields(prisma, userData, [formatted], partialInfo))[0] as RecursivePartial<GraphQLModel>;
    return result;
}

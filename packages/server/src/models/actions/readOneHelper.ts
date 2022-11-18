import { CODE, ViewFor } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { FindByIdOrHandleInput, FindByVersionInput } from "../../schema/types";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, getUser, modelToGraphQL, selectHelper, toPartialGraphQLInfo } from "../builder";
import { getAuthenticatedData, getIdFromHandle, getLatestVersion } from "../utils";
import { permissionsCheck } from "../validators";
import { ViewModel } from "../view";
import { ReadOneHelperProps } from "./types";

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    req,
}: ReadOneHelperProps<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>> {
    const objectType = model.format.relationshipMap.__typename;
    const userData = getUser(req);
    // Validate input. Can read by id, handle, or versionGroupId
    if (!input.id && !(input as FindByIdOrHandleInput).handle && !(input as FindByVersionInput).versionGroupId)
        throw new CustomError(CODE.InvalidArgs, 'id or handle is required', { code: genErrorCode('0019') });
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0020') });
    // If using versionGroupId, find the latest completed version in that group and use that id from now on
    let id: string | null | undefined;
    if ((input as FindByVersionInput).versionGroupId) {
        const versionId = await getLatestVersion({ objectType: objectType as any, prisma, versionGroupId: (input as FindByVersionInput).versionGroupId as string });
        id = versionId;
    }
    // If using handle, find the id of the object with that handle
    else if ((input as FindByIdOrHandleInput).handle) {
        id = await getIdFromHandle({ handle: (input as FindByIdOrHandleInput).handle as string, objectType: objectType as 'Organization' | 'Project' | 'User', prisma });
    } else {
        id = input.id;
    }
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.type]: [id] }, prisma, userData?.id ?? null);
    // Check permissions
    permissionsCheck(authDataById, { ['Read']: [id as string] }, userData);
    // Get the Prisma object
    let object = id ?
        await (model.prismaObject(prisma) as any).findUnique({ where: { id: id }, ...selectHelper(partialInfo) }) :
        await (model.prismaObject(prisma) as any).findFirst({ where: { handle: (input as FindByIdOrHandleInput).handle as string }, ...selectHelper(partialInfo) });
    if (!object)
        throw new CustomError(CODE.NotFound, `${objectType} not found`, { code: genErrorCode('0022') });
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, partialInfo) as RecursivePartial<GraphQLModel>;
    // If logged in and object has view count, handle it
    if (userData?.id && objectType in ViewFor) {
        ViewModel.mutate(prisma).view(userData.id, { forId: object.id, title: '', viewFor: objectType as any }); //TODO add title, which requires user's language
    }
    return (await addSupplementalFields(prisma, userData, [formatted], partialInfo))[0] as RecursivePartial<GraphQLModel>;
}
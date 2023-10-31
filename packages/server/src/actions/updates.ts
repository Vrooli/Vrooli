import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cuds";
import { UpdateManyHelperProps, UpdateOneHelperProps } from "./types";

/**
 * Helper function for updating multiple objects of the same type in a single line
 * @returns GraphQL response object
 */
export async function updateManyHelper<GraphQLModel>({
    info,
    input,
    objectType,
    prisma,
    req,
}: UpdateManyHelperProps): Promise<RecursivePartial<GraphQLModel>[]> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Get formatter and id field
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true);
    // Create objects. cudHelper will check permissions
    const updated = await cudHelper({
        inputData: input.map(d => ({
            actionType: "Update",
            input: d,
            objectType,
        })),
        partialInfo,
        prisma,
        userData,
    });
    // Make sure none of the items in the array are booleans
    if (updated.some(d => typeof d === "boolean")) {
        throw new CustomError("0032", "ErrorUnknown", userData.languages);
    }
    // Handle new version trigger, if applicable
    //TODO might be done in shapeUpdate. Not sure yet
    return await addSupplementalFields(prisma, userData, updated as Record<string, any>[], partialInfo) as RecursivePartial<GraphQLModel>[];
}

/**
 * Helper function for updating one object in a single line
 * @returns GraphQL response object
 */
export async function updateOneHelper<GraphQLModel>({
    input,
    ...rest
}: UpdateOneHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    return (await updateManyHelper<GraphQLModel>({ input: [input], ...rest }))[0];
}

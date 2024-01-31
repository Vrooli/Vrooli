import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cuds";
import { CreateManyHelperProps, CreateOneHelperProps } from "./types";

/**
 * Helper function for creating multiple objects of the same type in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createManyHelper<GraphQLModel>({
    info,
    input,
    objectType,
    prisma,
    req,
}: CreateManyHelperProps): Promise<RecursivePartial<GraphQLModel>[]> {
    const userData = assertRequestFrom(req, { isUser: true });
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true);
    // Create objects. cudHelper will check permissions
    const created = await cudHelper({
        inputData: input.map(d => ({
            action: "Create",
            input: d,
            objectType,
        })),
        partialInfo,
        prisma,
        userData,
    });
    // Make sure none of the items in the array are booleans
    if (created.some(d => typeof d === "boolean")) {
        throw new CustomError("0028", "ErrorUnknown", userData.languages);
    }
    return await addSupplementalFields(prisma, userData, created as Record<string, any>[], partialInfo) as RecursivePartial<GraphQLModel>[];
}

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createOneHelper<GraphQLModel>({
    input,
    ...rest
}: CreateOneHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    return (await createManyHelper<GraphQLModel>({ input: [input], ...rest }))[0];
}

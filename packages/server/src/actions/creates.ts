import { RequestService } from "../auth/request";
import { addSupplementalFields, InfoConverter } from "../builders/infoConverter";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cuds";
import { CreateManyHelperProps, CreateOneHelperProps } from "./types";

/**
 * Helper function for creating multiple objects of the same type in a single line.
 * Throws error if not successful.
 * @returns API response object
 */
export async function createManyHelper<ObjectModel>({
    additionalData,
    info,
    input,
    objectType,
    req,
}: CreateManyHelperProps): Promise<RecursivePartial<ObjectModel>[]> {
    const userData = RequestService.assertRequestFrom(req, { isUser: true });
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = InfoConverter.fromApiToPartialApi(info, format.apiRelMap, true);
    // Create objects. cudHelper will check permissions
    const created = await cudHelper({
        additionalData,
        info: partialInfo,
        inputData: input.map(d => ({
            action: "Create",
            input: d,
            objectType,
        })),
        userData,
    });
    // Make sure none of the items in the array are booleans
    if (created.some(d => typeof d === "boolean")) {
        throw new CustomError("0028", "ErrorUnknown");
    }
    return await addSupplementalFields(userData, created as Record<string, any>[], partialInfo) as RecursivePartial<ObjectModel>[];
}

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createOneHelper<ObjectModel>({
    input,
    ...rest
}: CreateOneHelperProps): Promise<RecursivePartial<ObjectModel>> {
    return (await createManyHelper<ObjectModel>({ input: [input], ...rest }))[0];
}

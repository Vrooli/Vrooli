import { SessionUser } from "@local/shared";
import { SessionService } from "../auth/session.js";
import { addSupplementalFields, InfoConverter } from "../builders/infoConverter.js";
import { CustomError } from "../events/error.js";
import { ModelMap } from "../models/base/index.js";
import { RecursivePartial } from "../types.js";
import { cudHelper } from "./cuds.js";
import { UpdateManyHelperProps, UpdateOneHelperProps } from "./types.js";

/**
 * Helper function for updating multiple objects of the same type in a single line
 * @returns GraphQL response object
 */
export async function updateManyHelper<ObjectModel>({
    adminFlags,
    additionalData,
    info,
    input,
    objectType,
    req,
}: UpdateManyHelperProps): Promise<RecursivePartial<ObjectModel>[]> {
    const userData = SessionService.getUser(req) as SessionUser;
    // Get formatter and id field
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = InfoConverter.get().fromApiToPartialApi(info, format.apiRelMap, true);
    // Create objects. cudHelper will check permissions
    const updated = await cudHelper({
        adminFlags,
        additionalData,
        info: partialInfo,
        inputData: input.map(d => ({
            action: "Update",
            input: d,
            objectType,
        })),
        userData,
    });
    // Make sure none of the items in the array are booleans
    if (updated.some(d => typeof d === "boolean")) {
        throw new CustomError("0032", "ErrorUnknown");
    }
    // Handle new version trigger, if applicable
    //TODO might be done in shapeUpdate. Not sure yet
    return await addSupplementalFields(userData, updated as Record<string, any>[], partialInfo) as RecursivePartial<ObjectModel>[];
}

/**
 * Helper function for updating one object in a single line
 * @returns GraphQL response object
 */
export async function updateOneHelper<ObjectModel>({
    input,
    ...rest
}: UpdateOneHelperProps): Promise<RecursivePartial<ObjectModel>> {
    return (await updateManyHelper<ObjectModel>({ input: [input], ...rest }))[0];
}

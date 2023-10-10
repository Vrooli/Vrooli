import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cuds";
import { CreateHelperProps } from "./types";

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createHelper<GraphQLModel>({
    info,
    input,
    objectType,
    prisma,
    req,
}: CreateHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    const userData = assertRequestFrom(req, { isUser: true });
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true);
    // Create objects. cudHelper will check permissions
    const created = (await cudHelper({
        inputData: [{
            actionType: "Create",
            input,
            objectType,
        }],
        partialInfo,
        prisma,
        userData,
    }))[0];
    if (typeof created !== "boolean") {
        return (await addSupplementalFields(prisma, userData, [created], partialInfo))[0] as any;
    }
    throw new CustomError("0028", "ErrorUnknown", userData.languages);
}

import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields } from "../builders/addSupplementalFields";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cuds";
import { UpdateHelperProps } from "./types";

/**
 * Helper function for updating one object in a single line
 * @returns GraphQL response object
 */
export async function updateHelper<GraphQLModel>({
    info,
    input,
    objectType,
    prisma,
    req,
}: UpdateHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Get formatter and id field
    const format = ModelMap.get(objectType).format;
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true);
    // Create objects. cudHelper will check permissions
    const updated = (await cudHelper({
        inputData: [{
            actionType: "Update",
            input,
            objectType,
        }],
        partialInfo,
        prisma,
        userData,
    }))[0];
    if (typeof updated !== "boolean") {
        // Handle new version trigger, if applicable
        //TODO might be done in shapeUpdate. Not sure yet
        return (await addSupplementalFields(prisma, userData, [updated], partialInfo))[0] as any;
    }
    throw new CustomError("0032", "ErrorUnknown", userData.languages);
}

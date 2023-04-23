import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cudHelper";
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
    where = (obj) => ({ id: obj.id }),
}: UpdateHelperProps): Promise<RecursivePartial<GraphQLModel>> {
    const userData = assertRequestFrom(req, { isUser: true });
    // Get formatter
    const { format } = getLogic(["format"], objectType, userData.languages, "cudHelper");
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.languages, true);
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: where(input), data: input };
    // Create objects. cudHelper will check permissions
    const cudResult = await cudHelper({ updateMany: [shapedInput], objectType, partialInfo, prisma, userData });
    const { updated } = cudResult;
    if (updated && updated.length > 0) {
        // Handle new version trigger, if applicable
        //TODO might be done in shapeUpdate. Not sure yet
        return (await addSupplementalFields(prisma, userData, updated, partialInfo))[0] as any;
    }
    throw new CustomError("0032", "ErrorUnknown", userData.languages);
}

import { assertRequestFrom } from "../auth/request";
import { addSupplementalFields, toPartialGqlInfo } from "../builders";
import { CustomError } from "../events";
import { getLogic } from "../getters";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cudHelper";
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
    const { format } = getLogic(["format"], objectType, req.session.languages, "createHelper");
    // Partially convert info type
    const partialInfo = toPartialGqlInfo(info, format.gqlRelMap, req.session.languages, true);
    // Create objects. cudHelper will check permissions
    const cudResult = await cudHelper({ createMany: [input], objectType, partialInfo, prisma, userData });
    const { created } = cudResult;
    if (created && created.length > 0) {
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0] as any;
    }
    throw new CustomError("0028", "ErrorUnknown", userData.languages);
}

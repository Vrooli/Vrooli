import { assertRequestFrom } from "../auth/request";
import { CustomError } from "../events";
import { RecursivePartial } from "../types";
import { cudHelper } from "./cudHelper";
import { CreateHelperProps } from "./types";
import { getFormatter } from "../getters";
import { addSupplementalFields, toPartialGraphQLInfo } from "../builders";

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
    const formatter = getFormatter(objectType, req.languages, 'createHelper');
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, formatter.relationshipMap, req.languages, true);
    // Create objects. cudHelper will check permissions
    const cudResult = await cudHelper({ createMany: [input], objectType, partialInfo, prisma, userData });
    const { created } = cudResult;
    if (created && created.length > 0) {
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0] as any;
    }
    throw new CustomError('0028', 'ErrorUnknown', userData.languages);
}
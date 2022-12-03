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
    console.log('create a');
    const userData = assertRequestFrom(req, { isUser: true });
    console.log('create b');
    const formatter = getFormatter(objectType, req.languages, 'createHelper');
    console.log('create c');
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, formatter.relationshipMap, req.languages, true);
    console.log('create d');
    // Create objects. cudHelper will check permissions
    const cudResult = await cudHelper({ createMany: [input], objectType, partialInfo, prisma, userData });
    console.log('create e');
    const { created } = cudResult;
    if (created && created.length > 0) {
        console.log('create f');
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0] as any;
    }
    console.log('create g');
    throw new CustomError('0028', 'ErrorUnknown', userData.languages);
}
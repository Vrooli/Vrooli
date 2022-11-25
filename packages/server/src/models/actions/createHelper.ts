import { assertRequestFrom } from "../../auth/request";
import { CustomError, Trigger } from "../../events";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, toPartialGraphQLInfo } from "../builder";
import { GraphQLModelType } from "../types";
import { getFormatter } from "../utils";
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
}: CreateHelperProps<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>> {
    const userData = assertRequestFrom(req, { isUser: true });
    const formatter = getFormatter(objectType, req.languages, 'createHelper');
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, formatter.relationshipMap);
    if (!partialInfo)
        throw new CustomError('0027', 'InternalError', userData.languages);
    // Create objects. cudHelper will check permissions
    const cudResult = await cudHelper({ createMany: [input], objectType, partialInfo, prisma, userData });
    const { created } = cudResult;
    if (created && created.length > 0) {
        const objectType = partialInfo.__typename as GraphQLModelType;
        // Handle trigger
        for (const id of input.ids) {
            await Trigger(prisma, req.languages).objectCreate(objectType, id, userData.id);
        }
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0] as any;
    }
    throw new CustomError('0028', 'ErrorUnknown', userData.languages);
}
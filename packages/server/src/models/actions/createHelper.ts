import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, getUserId, toPartialGraphQLInfo } from "../builder";
import { CreateHelperProps } from "./types";

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    req,
}: CreateHelperProps<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>> {
    const userId = getUserId(req);
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0025') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support create', { code: genErrorCode('0026') });
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0027') });
    // Create objects. cud will check permissions
    const cudResult = await model.mutate!(prisma).cud!({ partialInfo, userId, createMany: [input] });
    const { created } = cudResult;
    if (created && created.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partialInfo.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = created.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Create,
                object1Type: objectType,
                object1Id: c.id,
            }));
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Create" log', { code: genErrorCode('0194'), error }));
        }
        return (await addSupplementalFields(prisma, userId, created, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in createHelper', { code: genErrorCode('0028') });
}
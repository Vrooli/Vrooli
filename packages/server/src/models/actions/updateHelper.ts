import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, logger, LogLevel } from "../../events";
import { LogType } from "../../schema/types";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, getUserId, toPartialGraphQLInfo } from "../builder";
import { UpdateHelperProps } from "./types";

/**
 * Helper function for updating one object in a single line
 * @returns GraphQL response object
 */
export async function updateHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    req,
    where = (obj) => ({ id: obj.id }),
}: UpdateHelperProps<any>): Promise<RecursivePartial<GraphQLModel>> {
    const userId = getUserId(req);
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0029') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support update', { code: genErrorCode('0030') });
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0031') });
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: where(input), data: input };
    // Update objects. cud will check permissions
    const { updated } = await model.mutate!(prisma).cud!({ partialInfo, userId, updateMany: [shapedInput] });
    if (updated && updated.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partialInfo.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = updated.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Update,
                object1Type: objectType,
                object1Id: c.id,
            }));
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Update" log', { code: genErrorCode('0195'), error }));
        }
        return (await addSupplementalFields(prisma, userId, updated, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in updateHelper', { code: genErrorCode('0032') });
}
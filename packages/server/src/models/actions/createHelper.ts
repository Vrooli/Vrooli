import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, getUser, toPartialGraphQLInfo } from "../builder";
import { GraphQLModelType } from "../types";
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
    const userData = getUser(req);
    if (!userData)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0025') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support create', { code: genErrorCode('0026') });
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0027') });
    // Create objects. cud will check permissions
    const cudResult = await model.mutate!(prisma).cud!({ partialInfo, userData, createMany: [input] });
    const { created } = cudResult;
    if (created && created.length > 0) {
        const objectType = partialInfo.__typename as GraphQLModelType;
        // Handle trigger
        for (const id of input.ids) {
            await Trigger(prisma).objectCreate(objectType, id, userData.id);
        }
        return (await addSupplementalFields(prisma, userData.id, created, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in createHelper', { code: genErrorCode('0028') });
}
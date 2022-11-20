import { assertRequestFrom } from "../../auth/auth";
import { CustomError, Trigger } from "../../events";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, toPartialGraphQLInfo } from "../builder";
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
    const userData = assertRequestFrom(req, { isUser: true });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError('CreateNotSupported', { trace: '0026' });
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError('InternalError', { trace: '0027' });
    // Create objects. cud will check permissions
    const cudResult = await model.mutate!(prisma).cud!({ partialInfo, userData, createMany: [input] });
    const { created } = cudResult;
    if (created && created.length > 0) {
        const objectType = partialInfo.__typename as GraphQLModelType;
        // Handle trigger
        for (const id of input.ids) {
            await Trigger(prisma).objectCreate(objectType, id, userData.id);
        }
        return (await addSupplementalFields(prisma, userData, created, partialInfo))[0] as any;
    }
    throw new CustomError('ErrorUnknown', { trace: '0028' });
}
import { assertRequestFrom } from "../../auth/auth";
import { CustomError } from "../../events";
import { RecursivePartial } from "../../types";
import { addSupplementalFields, toPartialGraphQLInfo } from "../builder";
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
    const userData = assertRequestFrom(req, { isUser: true });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError('0030', 'UpdateNotSupported', userData.languages);
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError('0031', 'InternalError', userData.languages);
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: where(input), data: input };
    // Update objects. cud will check permissions
    const { updated } = await model.mutate!(prisma).cud!({ partialInfo, userData, updateMany: [shapedInput] });
    if (updated && updated.length > 0) {
        // Handle new version trigger, if applicable
        //TODO might be done in shapeUpdate. Not sure yet
        return (await addSupplementalFields(prisma, userData, updated, partialInfo))[0] as any;
    }
    throw new CustomError('0032', 'ErrorUnknown', userData.languages);
}
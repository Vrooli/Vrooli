import { CODE } from "@shared/consts";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { getUserId, toPartialGraphQLInfo, lowercaseFirstLetter } from "../builder";
import { readOneHelper } from "./readOneHelper";
import { ForkHelperProps } from "./types";

/**
 * Helper function for forking an object in a single line
 * @returns GraphQL Success response object
 */
export async function forkHelper({
    info,
    input,
    intendToPullRequest,
    model,
    prisma,
    req,
}: ForkHelperProps<any>): Promise<any> {
    const userId = getUserId(req);
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to fork object', { code: genErrorCode('0233') });
    if (!model.mutate || !model.mutate(prisma).duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support fork', { code: genErrorCode('0234') });
    // Check permissions
    const permissions: { [x: string]: any }[] = model.permissions ? await model.permissions().get({ objects: [{ id: input.id }], prisma, userId }) : [{}];
    if (!permissions[0].canFork && !permissions[0].canCopy) {
        throw new CustomError(CODE.Unauthorized, 'Not allowed to fork object', { code: genErrorCode('0262') });
    }
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, ({
        '__typename': 'Fork',
        'organization': 'Organization',
        'project': 'Project',
        'routine': 'Routine',
        'standard': 'Standard',
    }));
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0235') });
    const { object } = await model.mutate(prisma).duplicate!({ userId, objectId: input.id, isFork: false, createCount: 0 });
    // Handle trigger
    await Trigger(prisma).objectFork(input.objectType, input.id, userId);
    // Query for object
    const fullObject = await readOneHelper({
        info: (partialInfo as any)[lowercaseFirstLetter(input.objectType)],
        input: { id: object.id },
        model,
        prisma,
        req,
    })
    return fullObject;
}
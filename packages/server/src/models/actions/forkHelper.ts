import { CODE } from "@shared/consts";
import { assertRequestFrom } from "../../auth/auth";
import { CustomError, genErrorCode, Trigger } from "../../events";
import { toPartialGraphQLInfo, lowercaseFirstLetter } from "../builder";
import { getAuthenticatedData } from "../utils";
import { maxObjectsCheck, permissionsCheck } from "../validators";
import { readOneHelper } from "./readOneHelper";
import { ForkHelperProps } from "./types";

/**
 * Helper function for forking an object in a single line
 * @returns GraphQL Success response object
 */
export async function forkHelper({
    info,
    input,
    model,
    prisma,
    req,
}: ForkHelperProps<any>): Promise<any> {
    const userData = assertRequestFrom(req, { isUser: true });
    if (!model.mutate || !model.mutate(prisma).duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support fork', { code: genErrorCode('0234') });
    // Query for all authentication data
    const authDataById = await getAuthenticatedData({ [model.type]: [input.id] }, prisma, userData.id);
    // Check permissions
    await permissionsCheck(authDataById, { ['Create']: [input.id] }, userData);
    // Additional check for paywall
    //TODO
    // Check max objects
    await maxObjectsCheck(authDataById, { ['Create']: [input.id] }, prisma, userData);
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
    const { object } = await model.mutate(prisma).duplicate!({ userId: userData.id, objectId: input.id, isFork: false, createCount: 0 });
    // Handle trigger
    await Trigger(prisma).objectFork(input.objectType, input.id, userData.id);
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
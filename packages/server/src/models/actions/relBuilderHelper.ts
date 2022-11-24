import { relationshipBuilderHelper } from "../builder";
import { getMutater, getValidator } from "../utils";
import { RelBuilderHelperProps } from "./types";

/**
 * Helper function for calling appropriate relationship builder 
 * functions based on the operation being performed
 */
export async function relBuilderHelper<
    IsAdd extends boolean,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelName extends string,
    Input extends { [x: string]: any },
>({
    data,
    isAdd,
    isOneToOne,
    isRequired,
    objectType,
    prisma,
    relationshipName,
    userData,
}: RelBuilderHelperProps<IsAdd, IsOneToOne, IsRequired, RelName, Input>) {
    // Find validator and mutater for object type
    const validator = getValidator(objectType, userData.languages, 'relBuilderHelper');
    const mutater = getMutater(objectType, userData.languages, 'relBuilderHelper');
    return relationshipBuilderHelper<any, IsAdd, IsOneToOne, IsRequired, RelName, Input, any>({
        data,
        relationshipName,
        isAdd,
        isOneToOne,
        isRequired,
        isTransferable: validator.isTransferable,
        prisma,
        shape: mutater.shape as any,
        userData
    })
}
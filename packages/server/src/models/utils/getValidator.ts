import { CustomError } from "../../events";
import { ObjectMap } from "../builder";
import { GraphQLModelType, Validator } from "../types";

/**
 * Finds all permissions for the given object ids
 */
export function getValidator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    GQLModel extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends { [x: string]: any },
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any },
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Validator<GQLCreate, GQLUpdate, GQLModel, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere>{
    const validator: Validator<GQLCreate, GQLUpdate, GQLModel, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere> | undefined = ObjectMap[objectType]?.validate;
    if (!validator) {
        throw new CustomError('0280', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    return validator;
}
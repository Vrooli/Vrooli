import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { ObjectMap } from "../builder";
import { BasePermissions, GraphQLModelType, Validator } from "../types";

/**
 * Finds all permissions for the given object ids
 */
export function getValidator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    GQLModel extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends BasePermissions,
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any },
>(
    objectType: GraphQLModelType,
    errorTrace: string,
): Validator<GQLCreate, GQLUpdate, GQLModel, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere>{
    // Find validator and prisma delegate for this object type
    const validator: Validator<GQLCreate, GQLUpdate, GQLModel, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere> | undefined = ObjectMap[objectType]?.validate;
    if (!validator) {
        throw new CustomError(CODE.InvalidArgs, `Invalid object type in ${errorTrace}: ${objectType}`, { code: genErrorCode('0280') });
    }
    return validator;
}
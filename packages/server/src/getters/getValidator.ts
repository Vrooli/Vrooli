import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { GraphQLModelType, Validator } from "../models/types";

export function getValidator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends { [x: string]: any },
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any },
    IsTransferable extends boolean,
    IsVersioned extends boolean,
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Validator<GQLCreate, GQLUpdate, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere, IsTransferable, IsVersioned> {
    const validate = ObjectMap[objectType]?.validate;
    if (!validate) throw new CustomError('0280', 'InvalidArgs', languages, { errorTrace, objectType });
    return <Validator<GQLCreate, GQLUpdate, PrismaObject, PermissionObject, PermissionsSelect, OwnerOrMemberWhere, IsTransferable, IsVersioned>>validate;
}
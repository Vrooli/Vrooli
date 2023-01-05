import { Role, RoleCreateInput, RoleTranslation, RoleTranslationCreateInput, RoleTranslationUpdateInput, RoleUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "utils";

export type RoleTranslationShape = Pick<RoleTranslation, 'id' | 'language' | 'description'>

export type RoleShape = Pick<Role, 'id' | 'name' | 'permissions'> & {
    members?: { id: string }[];
    organization: { id: string };
    translations?: RoleTranslationShape[];
}

export const shapeRoleTranslation: ShapeModel<RoleTranslationShape, RoleTranslationCreateInput, RoleTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeRole: ShapeModel<RoleShape, RoleCreateInput, RoleUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'name', 'permissions'),
        ...createRel(d, 'members', ['Connect'], 'many'),
        ...createRel(d, 'organization', ['Connect'], 'one'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeRoleTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'name', 'permissions'),
        ...updateRel(o, u, 'members', ['Connect', 'Disconnect'], 'many'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeRoleTranslation),
    })
}
import { Organization, OrganizationCreateInput, OrganizationTranslation, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { MemberInviteShape, shapeMemberInvite } from "./memberInvite";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { RoleShape, shapeRole } from "./role";
import { shapeTag, TagShape } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type OrganizationTranslationShape = Pick<OrganizationTranslation, 'id' | 'language' | 'bio' | 'name'> & {
    __typename?: 'OrganizationTranslation';
}

export type OrganizationShape = Pick<Organization, 'id' | 'handle' | 'isOpenToNewMembers' | 'isPrivate'> & {
    __typename?: 'Organization';
    memberInvites?: MemberInviteShape[] | null;
    membersDelete?: { id: string }[] | null;
    resourceList?: ResourceListShape | null;
    roles?: RoleShape[] | null;
    tags?: ({ tag: string } | TagShape)[] | null;
    translations?: OrganizationTranslationShape[] | null;
}

export const shapeOrganizationTranslation: ShapeModel<OrganizationTranslationShape, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'bio', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'bio', 'name'), a)
}

export const shapeOrganization: ShapeModel<OrganizationShape, OrganizationCreateInput, OrganizationUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...createRel(d, 'memberInvites', ['Create'], 'many', shapeMemberInvite),
        ...createRel(d, 'resourceList', ['Create'], 'one', shapeResourceList),
        ...createRel(d, 'roles', ['Create'], 'many', shapeRole),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createRel(d, 'translations', ['Create'], 'many', shapeOrganizationTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...updateRel(o, u, 'memberInvites', ['Create', 'Delete'], 'many', shapeMemberInvite),
        ...updateRel(o, u, 'resourceList', ['Create', 'Update'], 'one', shapeResourceList),
        ...updateRel(o, u, 'roles', ['Create', 'Update', 'Delete'], 'many', shapeRole),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeOrganizationTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(m => m.id) } : {}),
    }, a)
}
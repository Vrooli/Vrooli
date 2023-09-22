import { Organization, OrganizationCreateInput, OrganizationTranslation, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { MemberInviteShape, shapeMemberInvite } from "./memberInvite";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { RoleShape, shapeRole } from "./role";
import { shapeTag, TagShape } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type OrganizationTranslationShape = Pick<OrganizationTranslation, "id" | "language" | "bio" | "name"> & {
    __typename?: "OrganizationTranslation";
}

export type OrganizationShape = Pick<Organization, "id" | "handle" | "isOpenToNewMembers" | "isPrivate"> & {
    __typename: "Organization";
    bannerImage?: string | File | null;
    memberInvites?: MemberInviteShape[] | null;
    membersDelete?: { id: string }[] | null;
    profileImage?: string | File | null;
    resourceList?: Omit<ResourceListShape, "listFor"> | null;
    roles?: RoleShape[] | null;
    tags?: ({ tag: string } | TagShape)[] | null;
    translations?: OrganizationTranslationShape[] | null;
}

export const shapeOrganizationTranslation: ShapeModel<OrganizationTranslationShape, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio", "name"), a),
};

export const shapeOrganization: ShapeModel<OrganizationShape, OrganizationCreateInput, OrganizationUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage");
        return {
            ...prims,
            ...createRel(d, "memberInvites", ["Create"], "many", shapeMemberInvite),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "Organization" } })),
            ...createRel(d, "roles", ["Create"], "many", shapeRole),
            ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
            ...createRel(d, "translations", ["Create"], "many", shapeOrganizationTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage"),
        ...updateRel(o, u, "memberInvites", ["Create", "Delete"], "many", shapeMemberInvite),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "Organization" } })),
        ...updateRel(o, u, "roles", ["Create", "Update", "Delete"], "many", shapeRole),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeOrganizationTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(m => m.id) } : {}),
    }, a),
};

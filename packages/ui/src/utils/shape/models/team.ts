import { Team, TeamCreateInput, TeamTranslation, TeamTranslationCreateInput, TeamTranslationUpdateInput, TeamUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { MemberInviteShape, shapeMemberInvite } from "./memberInvite";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { RoleShape, shapeRole } from "./role";
import { TagShape, shapeTag } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type TeamTranslationShape = Pick<TeamTranslation, "id" | "language" | "bio" | "name"> & {
    __typename?: "TeamTranslation";
}

export type TeamShape = Pick<Team, "id" | "handle" | "isOpenToNewMembers" | "isPrivate"> & {
    __typename: "Team";
    bannerImage?: string | File | null;
    memberInvites?: MemberInviteShape[] | null;
    membersDelete?: { id: string }[] | null;
    profileImage?: string | File | null;
    resourceList?: Omit<ResourceListShape, "listFor"> | null;
    roles?: RoleShape[] | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
    translations?: TeamTranslationShape[] | null;
}

export const shapeTeamTranslation: ShapeModel<TeamTranslationShape, TeamTranslationCreateInput, TeamTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "bio", "name"), a),
};

export const shapeTeam: ShapeModel<TeamShape, TeamCreateInput, TeamUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage");
        return {
            ...prims,
            ...createRel(d, "memberInvites", ["Create"], "many", shapeMemberInvite),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "Team" } })),
            ...createRel(d, "roles", ["Create"], "many", shapeRole),
            ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
            ...createRel(d, "translations", ["Create"], "many", shapeTeamTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "bannerImage", "handle", "isOpenToNewMembers", "isPrivate", "profileImage"),
        ...updateRel(o, u, "memberInvites", ["Create", "Delete"], "many", shapeMemberInvite),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "Team" } })),
        ...updateRel(o, u, "roles", ["Create", "Update", "Delete"], "many", shapeRole),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeTeamTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(m => m.id) } : {}),
    }, a),
};

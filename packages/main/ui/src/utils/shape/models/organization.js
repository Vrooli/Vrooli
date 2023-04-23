import { shapeMemberInvite } from "./memberInvite";
import { shapeResourceList } from "./resourceList";
import { shapeRole } from "./role";
import { shapeTag } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeOrganizationTranslation = {
    create: (d) => createPrims(d, "id", "language", "bio", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "bio", "name"), a),
};
export const shapeOrganization = {
    create: (d) => ({
        ...createPrims(d, "id", "handle", "isOpenToNewMembers", "isPrivate"),
        ...createRel(d, "memberInvites", ["Create"], "many", shapeMemberInvite),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList),
        ...createRel(d, "roles", ["Create"], "many", shapeRole),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createRel(d, "translations", ["Create"], "many", shapeOrganizationTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "handle", "isOpenToNewMembers", "isPrivate"),
        ...updateRel(o, u, "memberInvites", ["Create", "Delete"], "many", shapeMemberInvite),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList),
        ...updateRel(o, u, "roles", ["Create", "Update", "Delete"], "many", shapeRole),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeOrganizationTranslation),
        ...(u.membersDelete ? { membersDelete: u.membersDelete.map(m => m.id) } : {}),
    }, a),
};
//# sourceMappingURL=organization.js.map
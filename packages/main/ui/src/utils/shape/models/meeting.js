import { shapeMeetingInvite } from "./meetingInvite";
import { shapeSchedule } from "./schedule";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeMeetingTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "link", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "link", "name"), a),
};
export const shapeMeeting = {
    create: (d) => ({
        ...createPrims(d, "id", "openToAnyoneWithInvite", "showOnOrganizationProfile"),
        ...createRel(d, "organization", ["Connect"], "one"),
        ...createRel(d, "restrictedToRoles", ["Connect"], "many"),
        ...createRel(d, "invites", ["Create"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: d.id } })),
        ...createRel(d, "labels", ["Connect"], "many"),
        ...createRel(d, "schedule", ["Create"], "one", shapeSchedule),
        ...createRel(d, "translations", ["Create"], "many", shapeMeetingTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "openToAnyoneWithInvite", "showOnOrganizationProfile"),
        ...updateRel(o, u, "restrictedToRoles", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "invites", ["Create", "Update", "Delete"], "many", shapeMeetingInvite, (i) => ({ ...i, meeting: { id: o.id } })),
        ...updateRel(o, u, "labels", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "schedule", ["Create", "Update"], "one", shapeSchedule),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeMeetingTranslation),
    }, a),
};
//# sourceMappingURL=meeting.js.map
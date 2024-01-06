import { bool, description, id, name, opt, req, transRel, url, YupModel, yupObj } from "../utils";
import { meetingInviteValidation } from "./meetingInvite";
import { scheduleValidation } from "./schedule";

export const meetingTranslationValidation: YupModel<["create", "update"]> = transRel({
    create: (d) => ({
        name: opt(name),
        description: opt(description),
        link: opt(url(d)),
    }),
    update: (d) => ({
        name: opt(name),
        description: opt(description),
        link: opt(url(d)),
    }),
});

export const meetingValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnOrganizationProfile: opt(bool),
    }, [
        ["organization", ["Connect"], "one", "req"],
        ["restrictedToRoles", ["Connect"], "many", "opt"],
        ["invites", ["Create"], "many", "opt", meetingInviteValidation],
        ["labels", ["Connect"], "many", "opt"],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["translations", ["Create"], "many", "opt", meetingTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnOrganizationProfile: opt(bool),
    }, [
        ["restrictedToRoles", ["Connect", "Disconnect"], "many", "opt"],
        ["invites", ["Create", "Update", "Delete"], "many", "opt", meetingInviteValidation],
        ["labels", ["Connect", "Disconnect"], "many", "opt"],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", meetingTranslationValidation],
    ], [], d),
};

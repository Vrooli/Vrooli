import { bool, description, id, name, opt, req, transRel, url, YupModel, yupObj } from "../utils";
import { meetingInviteValidation } from "./meetingInvite";
import { scheduleValidation } from "./schedule";

export const meetingTranslationValidation: YupModel = transRel({
    create: {
        name: opt(name),
        description: opt(description),
        link: opt(url),
    },
    update: {
        name: opt(name),
        description: opt(description),
        link: opt(url),
    },
});

export const meetingValidation: YupModel = {
    create: ({ o }) => yupObj({
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
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnOrganizationProfile: opt(bool),
    }, [
        ["restrictedToRoles", ["Connect", "Disconnect"], "many", "opt"],
        ["invites", ["Create", "Update", "Delete"], "many", "opt"],
        ["labels", ["Connect", "Disconnect"], "many", "opt"],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["translations", ["Create"], "many", "opt", meetingTranslationValidation],
    ], [], o),
};

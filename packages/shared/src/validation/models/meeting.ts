import { opt, req } from "../utils/builders/optionality.js";
import { transRel } from "../utils/builders/rel.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, description, id, name, url } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";
import { meetingInviteValidation } from "./meetingInvite.js";
import { scheduleValidation } from "./schedule.js";

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
        showOnTeamProfile: opt(bool),
    }, [
        ["team", ["Connect"], "one", "req"],
        ["restrictedToRoles", ["Connect"], "many", "opt"],
        ["invites", ["Create"], "many", "opt", meetingInviteValidation],
        ["labels", ["Connect"], "many", "opt"],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["translations", ["Create"], "many", "opt", meetingTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnTeamProfile: opt(bool),
    }, [
        ["restrictedToRoles", ["Connect", "Disconnect"], "many", "opt"],
        ["invites", ["Create", "Update", "Delete"], "many", "opt", meetingInviteValidation],
        ["labels", ["Connect", "Disconnect"], "many", "opt"],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", meetingTranslationValidation],
    ], [], d),
};

/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in meeting.test.ts
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
        ["invites", ["Create"], "many", "opt", meetingInviteValidation],
        ["schedule", ["Create"], "one", "opt", scheduleValidation],
        ["translations", ["Create"], "many", "opt", meetingTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
        showOnTeamProfile: opt(bool),
    }, [
        ["invites", ["Create", "Update", "Delete"], "many", "opt", meetingInviteValidation],
        ["schedule", ["Create", "Update"], "one", "opt", scheduleValidation],
        ["translations", ["Create", "Update", "Delete"], "many", "opt", meetingTranslationValidation],
    ], [], d),
};

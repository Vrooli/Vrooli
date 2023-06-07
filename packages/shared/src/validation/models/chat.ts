import { bool, description, id, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { chatInviteValidation } from "./chatInvite";

export const chatTranslationValidation: YupModel = transRel({
    create: {
        name: opt(name),
        description: opt(description),
    },
    update: {
        name: opt(name),
        description: opt(description),
    },
});

export const chatValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create"], "many", "opt", chatInviteValidation],
        ["labels", ["Connect"], "many", "opt"],
        ["organization", ["Connect"], "one", "req"],
        ["translations", ["Create"], "many", "opt", chatTranslationValidation],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create", "Update", "Delete"], "many", "opt"],
        ["labels", ["Connect", "Disconnect"], "many", "opt"],
        ["participants", ["Delete"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", chatTranslationValidation],
    ], [], o),
};

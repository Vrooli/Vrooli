import { bool, description, id, name, opt, req, transRel, YupModel, yupObj } from "../utils";
import { chatInviteValidation } from "./chatInvite";
import { chatMessageValidation } from "./chatMessage";

export const chatTranslationValidation: YupModel = transRel({
    create: () => ({
        name: opt(name),
        description: opt(description),
    }),
    update: () => ({
        name: opt(name),
        description: opt(description),
    }),
});

export const chatValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create"], "many", "opt", chatInviteValidation, ["chat"]],
        ["labels", ["Connect"], "many", "opt"],
        ["messages", ["Create"], "many", "opt", chatMessageValidation, ["chat"]],
        ["organization", ["Connect"], "one", "opt"],
        ["translations", ["Create"], "many", "opt", chatTranslationValidation],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        openToAnyoneWithInvite: opt(bool),
    }, [
        ["invites", ["Create", "Update", "Delete"], "many", "opt", chatInviteValidation],
        ["labels", ["Connect", "Disconnect"], "many", "opt"],
        ["messages", ["Create", "Update", "Delete"], "many", "opt", chatMessageValidation],
        ["participants", ["Delete"], "many", "opt"],
        ["translations", ["Create"], "many", "opt", chatTranslationValidation],
    ], [], d),
};

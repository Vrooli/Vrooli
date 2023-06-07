import { id, message, opt, req, YupModel, yupObj } from "../utils";

export const chatInviteValidation: YupModel = {
    create: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [
        ["chat", ["Connect"], "one", "req"],
        ["user", ["Connect"], "many", "opt"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], o),
};

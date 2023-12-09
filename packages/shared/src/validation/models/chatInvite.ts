import { id, message, opt, req, YupModel, yupObj } from "../utils";

export const chatInviteValidation: YupModel = {
    create: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [
        ["chat", ["Connect"], "one", "req"],
        ["user", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};

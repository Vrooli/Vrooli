import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, message } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const chatInviteValidation: YupModel<["create", "update"]> = {
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

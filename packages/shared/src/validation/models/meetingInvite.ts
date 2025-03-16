import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, message } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const meetingInviteValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [
        ["meeting", ["Connect"], "one", "req"],
        ["user", ["Connect"], "many", "opt"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};

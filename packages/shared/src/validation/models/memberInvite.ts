import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, message, permissions } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const memberInviteValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [
        ["team", ["Connect"], "one", "req"],
        ["user", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [], [], d),
};

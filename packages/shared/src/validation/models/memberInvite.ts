import { bool, id, message, opt, permissions, req, YupModel, yupObj } from "../utils";

export const memberInviteValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [
        ["user", ["Connect"], "one", "req"],
        ["organization", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [], [], d),
};

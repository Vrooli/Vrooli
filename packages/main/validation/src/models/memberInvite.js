import { bool, id, message, opt, permissions, req, yupObj } from "../utils";
export const memberInviteValidation = {
    create: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [
        ["user", ["Connect"], "one", "req"],
        ["organization", ["Connect"], "one", "req"],
    ], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(bool),
        willHavePermissions: opt(permissions),
    }, [], [], o),
};
//# sourceMappingURL=memberInvite.js.map
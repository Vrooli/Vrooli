import { bool, id, opt, permissions, req, YupModel, yupObj } from "../utils";

export const memberValidation: YupModel<["update"]> = {
    // Can only be created through an invite
    update: (d) => yupObj({
        id: req(id),
        isAdmin: opt(bool),
        permissions: opt(permissions),
    }, [], [], d),
};

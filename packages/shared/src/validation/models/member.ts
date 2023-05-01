import { bool, id, opt, permissions, req, YupModel, yupObj } from "../utils";

export const memberValidation: YupModel<false, true> = {
    // Can only be created through an invite
    update: ({ o }) => yupObj({
        id: req(id),
        isAdmin: opt(bool),
        permissions: opt(permissions),
    }, [], [], o),
};

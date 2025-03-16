import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { bool, id, permissions } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const memberValidation: YupModel<["update"]> = {
    // Can only be created through an invite
    update: (d) => yupObj({
        id: req(id),
        isAdmin: opt(bool),
        permissions: opt(permissions),
    }, [], [], d),
};

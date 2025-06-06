/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in member.test.ts
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

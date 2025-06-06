/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in meetingInvite.test.ts
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
        ["user", ["Connect"], "one", "req"],
    ], [], d),
    update: (d) => yupObj({
        id: req(id),
        message: opt(message),
    }, [], [], d),
};

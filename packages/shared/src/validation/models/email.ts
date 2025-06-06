/* c8 ignore start */
// Coverage is ignored for validation model files because they export schema objects rather than
// executable functions. Their correctness is verified through comprehensive validation tests.
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { email, id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const emailValidation: YupModel<["create"]> = {
    create: (d) => yupObj({
        id: req(id),
        emailAddress: req(email),
    }, [], [], d),
    // Can't update an email. Push notifications & other email-related settings are updated elsewhere
};

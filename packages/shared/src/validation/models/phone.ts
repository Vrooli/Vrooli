/* c8 ignore start */
// Coverage is ignored for validation model files because they export schema objects rather than
// executable functions. Their correctness is verified through comprehensive validation tests.
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, phoneNumber } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const phoneValidation: YupModel<["create"]> = {
    create: (d) => yupObj({
        id: req(id),
        phoneNumber: req(phoneNumber),
    }, [], [], d),
    // Can't update an phone. Push notifications & other phone-related settings are updated elsewhere
};

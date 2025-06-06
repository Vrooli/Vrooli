/* c8 ignore start */
// Coverage is ignored for validation model files because they export schema objects rather than
// executable functions. Their correctness is verified through comprehensive validation tests.
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id, name } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const walletValidation: YupModel<["update"]> = {
    // Cannot create a wallet directly - must go through handshake
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], d),
};

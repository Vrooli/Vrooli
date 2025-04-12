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

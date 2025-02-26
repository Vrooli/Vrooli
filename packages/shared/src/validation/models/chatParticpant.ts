import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const chatParticipantValidation: YupModel<["update"]> = {
    update: (d) => yupObj({
        id: req(id),
    }, [], [], d),
};

/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in chatParticipant.test.ts
import { req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { id } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const chatParticipantValidation: YupModel<["update"]> = {
    update: (d) => yupObj({
        id: req(id),
    }, [], [], d),
};
/* c8 ignore stop */

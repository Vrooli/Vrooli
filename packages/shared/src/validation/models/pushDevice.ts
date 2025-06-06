/* c8 ignore start */
// This file contains validation schemas that are converted to functions by the yup library.
// The c8 coverage tool cannot accurately track coverage of these dynamically generated functions,
// so we exclude this file from coverage analysis. The validation logic is tested in pushDevice.test.ts
import { opt, req } from "../utils/builders/optionality.js";
import { yupObj } from "../utils/builders/yupObj.js";
import { doublePositiveOrZero, id, name, pushNotificationKeys, url } from "../utils/commonFields.js";
import { type YupModel } from "../utils/types.js";

export const pushDeviceValidation: YupModel<["create", "update"]> = {
    create: (d) => yupObj({
        endpoint: req(url(d)),
        expires: opt(doublePositiveOrZero),
        keys: req(pushNotificationKeys),
        name: opt(name),
    }, [], [], d),
    update: (d) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], d),
};

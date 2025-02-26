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

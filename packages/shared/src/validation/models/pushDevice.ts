import { doublePositiveOrZero, id, name, opt, pushNotificationKeys, req, url, YupModel, yupObj } from "../utils";

export const pushDeviceValidation: YupModel = {
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

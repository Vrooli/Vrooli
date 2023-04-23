import { doublePositiveOrZero, id, name, opt, pushNotificationKeys, req, url, yupObj } from "../utils";
export const pushDeviceValidation = {
    create: ({ o }) => yupObj({
        endpoint: req(url),
        expires: opt(doublePositiveOrZero),
        keys: req(pushNotificationKeys),
        name: opt(name),
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        name: opt(name),
    }, [], [], o),
};
//# sourceMappingURL=pushDevice.js.map
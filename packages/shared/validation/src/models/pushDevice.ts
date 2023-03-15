import { doublePositiveOrZero, id, name, opt, pushNotificationKeys, req, url, YupModel, yupObj } from '../utils';

export const pushDeviceValidation: YupModel = {
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
}
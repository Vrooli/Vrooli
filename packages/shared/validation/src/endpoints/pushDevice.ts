import { doublePositiveOrZero, id, name, opt, pushNotificationKeys, req, url, YupModel } from '../utils';
import * as yup from 'yup';

export const pushDeviceValidation: YupModel = {
    create: () => yup.object().shape({
        endpoint: req(url),
        expires: opt(doublePositiveOrZero),
        keys: req(pushNotificationKeys),
        name: opt(name),
    }),
    update: () => yup.object().shape({
        id: req(id),
        name: opt(name),
    }),
}
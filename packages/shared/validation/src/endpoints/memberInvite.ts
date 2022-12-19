import * as yup from 'yup';
import { id, message, opt, permissions, rel, req, YupModel } from "../utils";

export const memberInviteValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(yup.boolean()),
        willHavePermissions: opt(permissions),
        ...rel('user', ['Connect'], 'one', 'req'),
        ...rel('organization', ['Connect'], 'one', 'req'),
    }),
    update: () => yup.object().shape({
        id: req(id),
        message: opt(message),
        willBeAdmin: opt(yup.boolean()),
        willHavePermissions: opt(permissions),
    }),
}
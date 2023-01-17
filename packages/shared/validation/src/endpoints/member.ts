import { id, opt, permissions, req, YupModel } from '../utils';
import * as yup from 'yup';

export const memberValidation: YupModel<false, true> = {
    // Can only be created through an invite
    update: () => yup.object().shape({
        id: req(id),
        isAdmin: opt(yup.boolean()),
        permissions: opt(permissions),
    }),
}
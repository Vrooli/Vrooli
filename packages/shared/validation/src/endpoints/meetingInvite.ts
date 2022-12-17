import * as yup from 'yup';
import { id, message, opt, rel, req, YupModel } from "../utils";

export const meetingInviteValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        message: opt(message),
        ...rel('meeting', ['Connect'], 'one', 'req'),
        ...rel('user', ['Connect'], 'many', 'opt'),
    }),
    update: yup.object().shape({
        id: req(id),
        message: opt(message),
    }),

}
import { details, id, language, opt, rel, reportCreatedFor, reportReason, req, YupModel } from '../utils';
import * as yup from 'yup';

export const reportValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        createdFor: req(reportCreatedFor),
        details: opt(details),
        language: req(language),
        reason: req(reportReason),
        ...rel('createdFor', ['Connect'], 'one', 'req'),
    }),
    update: () => yup.object().shape({
        id: req(id),
        details: opt(details),
        language: opt(language),
        reason: opt(reportReason),
    }),
}
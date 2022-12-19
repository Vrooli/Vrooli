import * as yup from 'yup';
import { blankToUndefined, description, hexColor, id, maxStrErr, opt, rel, req, transRel, YupModel } from "../utils";

const label = yup.string().transform(blankToUndefined).max(128, maxStrErr)

export const labelTranslationValidation: YupModel = transRel({
    create: {
        description: req(description),
    },
    update: {
        description: opt(description),
    }
})

export const labelValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
        ...rel('organization', ['Connect'], 'one', 'opt'),
        ...rel('apis', ['Connect'], 'many', 'opt'),
        ...rel('issues', ['Connect'], 'many', 'opt'),
        ...rel('notes', ['Connect'], 'many', 'opt'),
        ...rel('projects', ['Connect'], 'many', 'opt'),
        ...rel('routines', ['Connect'], 'many', 'opt'),
        ...rel('smartContracts', ['Connect'], 'many', 'opt'),
        ...rel('standards', ['Connect'], 'many', 'opt'),
        ...rel('meetings', ['Connect'], 'many', 'opt'),
        ...rel('runProjectSchedules', ['Connect'], 'many', 'opt'),
        ...rel('runRoutineSchedules', ['Connect'], 'many', 'opt'),
        ...rel('translations', ['Create'], 'many', 'opt', labelTranslationValidation),
    }),
    update: () => yup.object().shape({
        id: req(id),
        label: req(label),
        color: opt(hexColor),
        ...rel('apis', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('issues', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('notes', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('projects', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('routines', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('smartContracts', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('standards', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('meetings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('runProjectSchedules', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('runRoutineSchedules', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', labelTranslationValidation),
    }),
}
import { description, id, name, minNumErr, blankToUndefined, req, opt, rel, YupModel, transRel } from '../utils';
import * as yup from 'yup';
import { nodeLoopValidation } from './nodeLoop';
import { nodeEndValidation } from './nodeEnd';
import { nodeRoutineListValidation } from './nodeRoutineList';

const columnIndex = yup.number().integer().min(0, minNumErr).nullable()
const rowIndex = yup.number().integer().min(0, minNumErr).nullable()
const type = yup.string().transform(blankToUndefined).oneOf([
    'End', 
    'Loop', 
    'RoutineList', 
    'Start'
])

export const nodeTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        name: req(name),
    },
    update: {
        description: opt(description),
        name: opt(name),
    }
})

export const nodeValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        columnIndex: opt(columnIndex),
        rowIndex: opt(rowIndex),
        type: req(type),
        ...rel('loop', ['Create'], 'one', 'opt', nodeLoopValidation),
        ...rel('nodeEnd', ['Create'], 'one', 'opt', nodeEndValidation),
        ...rel('nodeRoutineList', ['Create'], 'one', 'opt', nodeRoutineListValidation),
        ...rel('routineVersion', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', nodeTranslationValidation),
    }, [['loopCreate', 'nodeEndCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        columnIndex: opt(columnIndex),
        rowIndex: opt(rowIndex),
        type: opt(type),
        ...rel('loop', ['Create', 'Update', 'Delete'], 'one', 'opt', nodeLoopValidation),
        ...rel('nodeEnd', ['Update'], 'one', 'opt', nodeEndValidation),
        ...rel('nodeRoutineList', ['Update'], 'one', 'opt', nodeRoutineListValidation),
        ...rel('routineVersion', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeTranslationValidation),
    })
}
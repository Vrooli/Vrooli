import { description, id, name, minNumErr, blankToUndefined, req, opt, rel, YupModel, transRel } from '../utils';
import * as yup from 'yup';
import { nodeLoopValidation } from './nodeLoop';
import { nodeEndValidation } from './nodeEnd';
import { nodeRoutineListValidation } from './nodeRoutineList';

const columnIndex = yup.number().integer().min(0, minNumErr).nullable()
const rowIndex = yup.number().integer().min(0, minNumErr).nullable()
const nodeType = yup.string().transform(blankToUndefined).oneOf([
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
        nodeType: req(nodeType),
        rowIndex: opt(rowIndex),
        ...rel('end', ['Create'], 'one', 'opt', nodeEndValidation),
        ...rel('loop', ['Create'], 'one', 'opt', nodeLoopValidation),
        ...rel('routineList', ['Create'], 'one', 'opt', nodeRoutineListValidation),
        ...rel('routineVersion', ['Connect'], 'one', 'req'),
        ...rel('translations', ['Create'], 'many', 'opt', nodeTranslationValidation),
    }, [['endCreate', 'routineListCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: opt(nodeType),
        rowIndex: opt(rowIndex),
        ...rel('end', ['Update'], 'one', 'opt', nodeEndValidation),
        ...rel('loop', ['Create', 'Update', 'Delete'], 'one', 'opt', nodeLoopValidation),
        ...rel('routineList', ['Update'], 'one', 'opt', nodeRoutineListValidation),
        ...rel('routineVersion', ['Connect'], 'one', 'opt'),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeTranslationValidation),
    })
}
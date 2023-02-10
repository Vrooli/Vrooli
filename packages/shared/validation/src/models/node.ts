import { description, id, name, minNumErr, req, opt, YupModel, transRel, yupObj, enumToYup } from '../utils';
import * as yup from 'yup';
import { nodeLoopValidation } from './nodeLoop';
import { nodeEndValidation } from './nodeEnd';
import { nodeRoutineListValidation } from './nodeRoutineList';
import { NodeType } from '@shared/consts';

const columnIndex = yup.number().integer().min(0, minNumErr).nullable();
const rowIndex = yup.number().integer().min(0, minNumErr).nullable();
const nodeType = enumToYup(NodeType);

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
    create: ({ o }) => yupObj({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: req(nodeType),
        rowIndex: opt(rowIndex),
    }, [
        ['end', ['Create'], 'one', 'opt', nodeEndValidation],
        ['loop', ['Create'], 'one', 'opt', nodeLoopValidation],
        ['routineList', ['Create'], 'one', 'opt', nodeRoutineListValidation],
        ['routineVersion', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', nodeTranslationValidation],
    ], [['endCreate', 'routineListCreate']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: opt(nodeType),
        rowIndex: opt(rowIndex),
    }, [
        ['end', ['Update'], 'one', 'opt', nodeEndValidation],
        ['loop', ['Create', 'Update', 'Delete'], 'one', 'opt', nodeLoopValidation],
        ['routineList', ['Update'], 'one', 'opt', nodeRoutineListValidation],
        ['routineVersion', ['Connect'], 'one', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeTranslationValidation],
    ], [], o),
}
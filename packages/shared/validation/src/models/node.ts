import { NodeType } from '@shared/consts';
import * as yup from 'yup';
import { description, enumToYup, id, minNumErr, name, opt, req, transRel, YupModel, yupObj } from '../utils';
import { nodeEndValidation } from './nodeEnd';
import { nodeLoopValidation } from './nodeLoop';
import { nodeRoutineListValidation } from './nodeRoutineList';

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
        ['end', ['Create'], 'one', 'opt', nodeEndValidation, ['node']],
        ['loop', ['Create'], 'one', 'opt', nodeLoopValidation, ['node']],
        ['routineList', ['Create'], 'one', 'opt', nodeRoutineListValidation, ['node']],
        ['routineVersion', ['Connect'], 'one', 'req'],
        ['translations', ['Create'], 'many', 'opt', nodeTranslationValidation],
    ], [['endCreate', 'routineListCreate']], o),
    update: ({ o }) => yupObj({
        id: req(id),
        columnIndex: opt(columnIndex),
        nodeType: opt(nodeType),
        rowIndex: opt(rowIndex),
    }, [
        ['end', ['Update'], 'one', 'opt', nodeEndValidation, ['node']],
        ['loop', ['Create', 'Update', 'Delete'], 'one', 'opt', nodeLoopValidation, ['node']],
        ['routineList', ['Update'], 'one', 'opt', nodeRoutineListValidation, ['node']],
        ['routineVersion', ['Connect'], 'one', 'opt'],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', nodeTranslationValidation],
    ], [], o),
}
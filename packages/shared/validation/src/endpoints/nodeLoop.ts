import { id, maxStrErr, minStrErr, req, opt, rel, YupModel, nodeOperation } from '../utils';
import * as yup from 'yup';
import { nodeLoopWhileValidation } from './nodeLoopWhile';

const loops = yup.number().integer().min(0, minStrErr).max(100, maxStrErr)
const maxLoops = yup.number().integer().min(1, minStrErr).max(100, maxStrErr)

export const nodeLoopValidation: YupModel = {
    create: yup.object().shape({
        id: req(id),
        loops: opt(loops),
        maxLoops: opt(maxLoops),
        operation: opt(nodeOperation),
        ...rel('whiles', ['Create'], 'many', 'req', nodeLoopWhileValidation),
    }),
    update: yup.object().shape({
        id: req(id),
        loops: opt(loops),
        maxLoops: opt(maxLoops),
        operation: opt(nodeOperation),
        ...rel('whiles', ['Create', 'Update', 'Delete'], 'many', 'req', nodeLoopWhileValidation),
    })
}
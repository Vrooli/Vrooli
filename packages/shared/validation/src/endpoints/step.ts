import { blankToUndefined, id, minNumErr, name, opt, req } from '../utils';
import * as yup from 'yup';
import { RunRoutineStepStatus } from '@shared/consts';

const order = yup.number().integer().min(0, minNumErr);
const contextSwitches = yup.number().integer().min(0, minNumErr);
const stepStatus = yup.string().transform(blankToUndefined).oneOf(Object.values(RunRoutineStepStatus))
const timeElapsed = yup.number().integer().min(0, minNumErr);
const step = yup.array().of(yup.number().integer().min(0, minNumErr));

export const stepCreate = yup.object().shape({
    id: req(id),
    nodeId: req(id),
    order: req(order),
    step: req(step),
    name: req(name),
})

export const stepUpdate = yup.object().shape({
    id: req(id),
    contextSwitches: opt(contextSwitches),
    status: opt(stepStatus),
    timeElapsed: opt(timeElapsed),
})
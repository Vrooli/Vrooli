import { id, minNumberErrorMessage, requiredErrorMessage, title } from './base';
import * as yup from 'yup';
import { RunStepStatus } from '@shared/consts';

const order = yup.number().integer().min(0, minNumberErrorMessage);
const contextSwitches = yup.number().integer().min(0, minNumberErrorMessage);
const stepStatus = yup.string().oneOf(Object.values(RunStepStatus))
const timeElapsed = yup.number().integer().min(0, minNumberErrorMessage);
const step = yup.array().of(yup.number().integer().min(0, minNumberErrorMessage));

export const stepCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    nodeId: id.required(requiredErrorMessage),
    order: order.required(requiredErrorMessage),
    step: step.required(requiredErrorMessage),
    title: title.required(requiredErrorMessage),
})

export const stepUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    contextSwitches: contextSwitches.notRequired().default(undefined),
    status: stepStatus.notRequired().default(undefined),
    timeElapsed: timeElapsed.notRequired().default(undefined),
})

export const stepsCreate = yup.array().of(stepCreate.required(requiredErrorMessage))
export const stepsUpdate = yup.array().of(stepUpdate.required(requiredErrorMessage))
import { id, title } from './base';
import * as yup from 'yup';
import { RunStepStatus } from '../consts';

const order = yup.number().integer().min(0);
const pickups = yup.number().integer().min(0);
const stepStatus = yup.string().oneOf(Object.values(RunStepStatus))
const timeElapsed = yup.number().integer().min(0);

export const stepCreate = yup.object().shape({
    order: order.required(),
    title: title.required(),
})

export const stepUpdate = yup.object().shape({
    id: id.required(),
    pickups: pickups.notRequired().default(undefined),
    status: stepStatus.notRequired().default(undefined),
    timeElapsed: timeElapsed.notRequired().default(undefined),
})

export const stepsCreate = yup.array().of(stepCreate.required())
export const stepsUpdate = yup.array().of(stepUpdate.required())
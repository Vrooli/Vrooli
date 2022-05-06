import { id, title } from './base';
import * as yup from 'yup';

const order = yup.number().integer().min(0);
const pickups = yup.number().integer().min(0);
const timeElapsed = yup.number().integer().min(0);

export const stepCreate = yup.object().shape({
    order: order.required(),
    title: title.required(),
})

export const stepUpdate = yup.object().shape({
    id: id.required(),
    pickups: pickups.notRequired().default(undefined),
    timeElapsed: timeElapsed.notRequired().default(undefined),
})

export const stepComplete = yup.object().shape({
    id: id.required(),
    endNodeId: id.required(),
})

export const stepSkip = yup.object().shape({
    id: id.required(),
})

export const stepsCreate = yup.array().of(stepCreate.required())
export const stepsUpdate = yup.array().of(stepUpdate.required())
export const stepsComplete = yup.array().of(stepComplete.required())
export const stepsSkip = yup.array().of(stepSkip.required())
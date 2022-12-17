import { id, idArray, minNumErr, name, opt, req } from '../utils';
import * as yup from 'yup';

const completedComplexity = yup.number().integer().min(0, minNumErr);
const contextSwitches = yup.number().integer().min(0, minNumErr);
const timeElapsed = yup.number().integer().min(0, minNumErr);
const isPrivate = yup.boolean();

export const runCreate = yup.object().shape({
    id: req(id),
    isPrivate: opt(isPrivate),
    routineId: req(id),
    name: req(name),
    // version: req(version()),
})

export const runUpdate = yup.object().shape({
    id: req(id),
    completedComplexity: opt(completedComplexity),
    contextSwitches: opt(contextSwitches),
    isPrivate: opt(isPrivate),
    timeElapsed: opt(timeElapsed),
    // stepsCreate: opt(stepsCreate),
    // stepsUpdate: opt(stepsUpdate),
    // stepsDelete: opt(idArray),
    // inputsCreate: opt(runInputsCreate),
    // inputsUpdate: opt(runInputsUpdate),
    inputsDelete: opt(idArray),
})
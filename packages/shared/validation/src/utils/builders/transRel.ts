import * as yup from 'yup';
import { req } from './req';
import { opt } from './opt';
import { id, language } from '../commonFields';
import { YupModel } from "../../utils/types";
import { yupObj } from './yupObj';

/**
 * Builds a YupModel for a translation object. 
 * All translation objects function the same way: 
 * - They have an id and language field
 * - They have additional, optional or required fields
 * @param parialYupModel Partial YupModel for the translation object. Only 
 * includes the additional fields
 * @returns YupModel for the translation object
 */
export const transRel = (partialYupModel: ({
    create: { [key: string]: yup.StringSchema }
    update: { [key: string]: yup.StringSchema }
})): YupModel => ({
    create: ({ o }) => yupObj({
        id: req(id),
        language: req(language),
        ...partialYupModel.create
    }, [], [], o),
    update: ({ o }) => yupObj({
        id: req(id),
        language: opt(language),
        ...partialYupModel.update
    }, [], [], o),
})
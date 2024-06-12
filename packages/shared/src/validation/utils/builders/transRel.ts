import * as yup from "yup";
import { YupModel, YupMutateParams } from "../../utils/types";
import { id, language } from "../commonFields";
import { opt } from "./opt";
import { req } from "./req";
import { yupObj } from "./yupObj";

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
    create: (params: YupMutateParams) => { [key: string]: yup.StringSchema };
    update: (params: YupMutateParams) => { [key: string]: yup.StringSchema };
})): YupModel<["create", "update"]> => ({
    create: (data) => yupObj({
        id: req(id),
        language: req(language),
        ...partialYupModel.create(data),
    }, [], [], data),
    update: (data) => yupObj({
        id: req(id),
        language: opt(language),
        ...partialYupModel.update(data),
    }, [], [], data),
});

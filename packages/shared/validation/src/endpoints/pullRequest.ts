import * as yup from 'yup';
import { YupModel } from "../utils";

export const pullRequestValidation: YupModel = {
    create: () => yup.object().shape({
    }),
    update: () => yup.object().shape({
    }),
}
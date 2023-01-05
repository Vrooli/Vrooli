import * as yup from 'yup';
import { YupModel } from "../utils";

export const smartContractVersionValidation: YupModel = {
    create: () => yup.object().shape({
    }),
    update: () => yup.object().shape({
    }),
}
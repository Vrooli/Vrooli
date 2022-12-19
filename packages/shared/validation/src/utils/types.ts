import * as yup from 'yup';

export type YupModel = {
    create?: () => yup.ObjectSchema<any>;
    update?: () => yup.ObjectSchema<any>;
}
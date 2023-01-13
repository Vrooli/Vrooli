import * as yup from 'yup';

export type YupModel<
    HasCreate extends boolean = true,
    HasUpdate extends boolean = true,
> = (HasCreate extends true ? {
    create: (...params: any) => yup.ObjectSchema<any>;
} : {}) & (HasUpdate extends true ? {
    update: (...params: any) => yup.ObjectSchema<any>;
} : {});
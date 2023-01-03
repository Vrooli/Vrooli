import * as yup from 'yup';

export type YupModel<
    HasCreate extends boolean = true,
    HasUpdate extends boolean = true,
> = (HasCreate extends true ? {
    create: () => yup.ObjectSchema<any>;
} : {}) & (HasUpdate extends true ? {
    update: () => yup.ObjectSchema<any>;
} : {});
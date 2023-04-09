import * as yup from 'yup';

export type YupModel<
    HasCreate extends boolean = true,
    HasUpdate extends boolean = true,
> = (HasCreate extends true ? {
    create: (params: { o?: string | string[] | undefined, minVersion?: string }) => yup.ObjectSchema<any>;
} : {}) & (HasUpdate extends true ? {
    update: (params: { o?: string | string[] | undefined, minVersion?: string }) => yup.ObjectSchema<any>;
} : {});
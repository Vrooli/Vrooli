import * as yup from "yup";

export type YupMutateParams = {
    /** Relationships to omit from the yup object */
    omitRels?: string | string[] | undefined,
    minVersion?: string,
    env?: "development" | "production",
}

export type YupModel<
    HasCreate extends boolean = true,
    HasUpdate extends boolean = true,
> = (HasCreate extends true ? {
    create: (params: YupMutateParams) => yup.ObjectSchema<any>;
} : {}) & (HasUpdate extends true ? {
    update: (params: YupMutateParams) => yup.ObjectSchema<any>;
} : {});

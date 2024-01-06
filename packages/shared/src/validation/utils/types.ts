import * as yup from "yup";

export type YupMutateParams = {
    /** Relationships to omit from the yup object */
    omitRels?: string | string[] | undefined,
    minVersion?: string,
    env?: string | undefined, // expect "production" | "prod" | "development" | "dev"
}

type ObjectShape = any;

export type YupModelOptions = "create" | "update" | "read";

type YupModelMethods = {
    create: (params: YupMutateParams) => yup.ObjectSchema<ObjectShape>;
    update: (params: YupMutateParams) => yup.ObjectSchema<ObjectShape>;
    read: (params: YupMutateParams) => yup.ObjectSchema<ObjectShape>;
};

type ElementType<T extends ReadonlyArray<unknown>> = T extends ReadonlyArray<infer ElementType> ? ElementType : never;

export type YupModel<T extends YupModelOptions[]> = {
    [P in ElementType<T>]: YupModelMethods[P];
};

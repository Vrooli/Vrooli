import * as yup from "yup";

export type YupMutateParams = {
    /** 
     * Relationships to omit from the yup object. 
     * Supports dot notation for nested relationships.
     */
    omitFields?: string[] | undefined,
    minVersion?: string,
    env?: string | undefined, // expect "production" | "prod" | "development" | "dev"
}

export type YupModelOptions = "create" | "update" | "read";

type YupModelMethods = {
    create: (params: YupMutateParams) => yup.AnySchema;
    update: (params: YupMutateParams) => yup.AnySchema;
    read: (params: YupMutateParams) => yup.AnySchema;
};

type ElementType<T extends ReadonlyArray<unknown>> = T extends ReadonlyArray<infer ElementType> ? ElementType : never;

export type YupModel<T extends YupModelOptions[]> = {
    [P in ElementType<T>]: YupModelMethods[P];
};

/* c8 ignore start */
import type * as yup from "yup";

export type YupMutateParams = {
    /** 
     * Relationships to omit from the yup object. 
     * Supports dot notation for nested relationships.
     */
    omitFields?: string[] | undefined,
    minVersion?: string,
    /** Tracks recursion to cancel early */
    recurseCount?: number,
    env?: string | undefined, // expect "production" | "prod" | "development" | "dev"
}

export type YupModelOptions = "create" | "update" | "read";

type YupModelMethods = {
    create: (params: YupMutateParams) => yup.AnyObjectSchema;
    update: (params: YupMutateParams) => yup.AnyObjectSchema;
    read: (params: YupMutateParams) => yup.AnyObjectSchema;
};

type ElementType<T extends ReadonlyArray<unknown>> = T extends ReadonlyArray<infer ElementType> ? ElementType : never;

export type YupModel<T extends YupModelOptions[]> = {
    [P in ElementType<T>]: YupModelMethods[P];
};

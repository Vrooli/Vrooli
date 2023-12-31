import { YupModel } from "@local/shared";
import { validateAndGetYupErrors } from "./shape/general";

type TransformFunction<T, U, IsCreate extends boolean> = (values: T, existing: U, isCreate: IsCreate) =>
    IsCreate extends true ? object : (object | undefined);

export const validateFormValues = async <T, IsCreate extends boolean>(
    values: T,
    existing: T,
    isCreate: IsCreate,
    transformFunction: TransformFunction<T, T, IsCreate>,
    validationMap: YupModel<true, true>,
): Promise<Record<string, string>> => {
    const transformedValues = transformFunction(values, existing, isCreate);
    const env = process.env.PROD ? "production" : "development";
    const validationSchema = validationMap[isCreate ? "create" : "update"]({ env });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

import { YupModel } from "@local/shared";
import { FormErrors } from "types";
import { validateAndGetYupErrors } from "./shape/general";

type TransformFunction<T, U, IsCreate extends boolean> = (values: T, existing: U, isCreate: IsCreate) =>
    IsCreate extends true ? object : (object | undefined);

export const validateFormValues = async <T, IsCreate extends boolean>(
    values: T,
    existing: T,
    isCreate: IsCreate,
    transformFunction: TransformFunction<T, T, IsCreate>,
    validationMap: YupModel<["create", "update"]>,
): Promise<FormErrors> => {
    const transformedValues = transformFunction(values, existing, isCreate);
    // If transformed values is empty (or only has an ID) and we're updating, create "No changes" error
    if (!isCreate && (transformedValues === undefined || Object.keys(transformedValues).length <= 1)) {
        return { _form: "No changes" };
    }
    const validationSchema = validationMap[isCreate ? "create" : "update"]({ env: process.env.NODE_ENV });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

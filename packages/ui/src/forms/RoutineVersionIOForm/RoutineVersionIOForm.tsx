import { DUMMY_ID, orDefault, RoutineVersionInput, routineVersionInputValidation, RoutineVersionOutput, routineVersionOutputValidation, Session, uuid } from "@local/shared";
import { getUserLanguages } from "utils/display/translationTools";
import { RoutineVersionInputShape, shapeRoutineVersionInput } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape, shapeRoutineVersionOutput } from "utils/shape/models/routineVersionOutput";
import { validateFormValues } from "utils/validateFormValues";

export const routineVersionInputInitialValues = (
    session: Session | undefined,
    existing?: RoutineVersionInput | null | undefined,
): RoutineVersionInputShape => ({
    __typename: "RoutineVersionInput" as const,
    id: uuid(), // Cannot be a dummy ID because routine versions reference this ID
    index: existing?.index ?? 0,
    isRequired: true,
    name: `Input ${(existing?.index ?? 0) + 1}`,
    routineVersion: existing!.routineVersion,
    standardVersion: existing?.standardVersion ?? null,
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "RoutineVersionInputTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        helpText: "",
    }]),
});

export const routineVersionOutputInitialValues = (
    session: Session | undefined,
    existing?: RoutineVersionOutput | null | undefined,
): RoutineVersionOutputShape => ({
    __typename: "RoutineVersionOutput" as const,
    id: uuid(), // Cannot be a dummy ID because routine versions reference this ID
    index: existing?.index ?? 0,
    name: `Output ${(existing?.index ?? 0) + 1}`,
    routineVersion: existing!.routineVersion,
    standardVersion: existing?.standardVersion ?? null,
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "RoutineVersionOutputTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        helpText: "",
    }]),
});

export const routineVersionIOInitialValues = <IsInput extends boolean>(
    session: Session | undefined,
    isInput: IsInput,
    existing?: RoutineVersionInput | RoutineVersionOutput | null | undefined,
): IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape => {
    if (isInput) {
        return routineVersionInputInitialValues(session, existing as RoutineVersionInput | undefined) as any;
    } else {
        return routineVersionOutputInitialValues(session, existing as RoutineVersionOutput | undefined) as any;
    }
};

export const transformRoutineVersionInputValues = (values: RoutineVersionInputShape, existing: RoutineVersionInputShape, isCreate: boolean) =>
    isCreate ? shapeRoutineVersionInput.create(values) : shapeRoutineVersionInput.update(existing, values);

export const transformRoutineVersionOutputValues = (values: RoutineVersionOutputShape, existing: RoutineVersionOutputShape, isCreate: boolean) =>
    isCreate ? shapeRoutineVersionOutput.create(values) : shapeRoutineVersionOutput.update(existing, values);

export const transformRoutineVersionIOValues = <IsInput extends boolean>(
    values: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isInput: IsInput,
    existing: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isCreate: boolean,
): IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape => {
    if (isInput) {
        return transformRoutineVersionInputValues(values as RoutineVersionInputShape, existing as RoutineVersionInputShape, isCreate) as any;
    } else {
        return transformRoutineVersionOutputValues(values as RoutineVersionOutputShape, existing as RoutineVersionOutputShape, isCreate) as any;
    }
};

export const validateRoutineVersionIOValues = async <IsInput extends boolean>(
    values: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isInput: IsInput,
    existing: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isCreate: boolean,
) => {
    if (isInput) {
        return validateFormValues(values as RoutineVersionInputShape, existing as RoutineVersionInputShape, isCreate, transformRoutineVersionInputValues, routineVersionInputValidation);
    } else {
        return validateFormValues(values as RoutineVersionOutputShape, existing as RoutineVersionOutputShape, isCreate, transformRoutineVersionOutputValues, routineVersionOutputValidation);
    }
};

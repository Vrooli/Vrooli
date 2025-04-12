import { DUMMY_ID, orDefault, RoutineVersionInput, RoutineVersionInputShape, routineVersionInputValidation, RoutineVersionOutput, RoutineVersionOutputShape, routineVersionOutputValidation, Session, shapeRoutineVersionInput, shapeRoutineVersionOutput, uuid } from "@local/shared";
import { getUserLanguages } from "../../utils/display/translationTools.js";
import { validateFormValues } from "../../utils/validateFormValues.js";

export function routineVersionInputInitialValues(
    session: Session | undefined,
    existing?: RoutineVersionInput | null | undefined,
): RoutineVersionInputShape {
    return {
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
    };
}

export function routineVersionOutputInitialValues(
    session: Session | undefined,
    existing?: RoutineVersionOutput | null | undefined,
): RoutineVersionOutputShape {
    return {
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
    };
}

export function routineVersionIOInitialValues<IsInput extends boolean>(
    session: Session | undefined,
    isInput: IsInput,
    existing?: RoutineVersionInput | RoutineVersionOutput | null | undefined,
): IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape {
    if (isInput) {
        return routineVersionInputInitialValues(session, existing as RoutineVersionInput | undefined) as any;
    } else {
        return routineVersionOutputInitialValues(session, existing as RoutineVersionOutput | undefined) as any;
    }
}

export function transformRoutineVersionInputValues(values: RoutineVersionInputShape, existing: RoutineVersionInputShape, isCreate: boolean) {
    return isCreate ? shapeRoutineVersionInput.create(values) : shapeRoutineVersionInput.update(existing, values);
}

export function transformRoutineVersionOutputValues(values: RoutineVersionOutputShape, existing: RoutineVersionOutputShape, isCreate: boolean) {
    return isCreate ? shapeRoutineVersionOutput.create(values) : shapeRoutineVersionOutput.update(existing, values);
}

export function transformRoutineVersionIOValues<IsInput extends boolean>(
    values: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isInput: IsInput,
    existing: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isCreate: boolean,
): IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape {
    if (isInput) {
        return transformRoutineVersionInputValues(values as RoutineVersionInputShape, existing as RoutineVersionInputShape, isCreate) as any;
    } else {
        return transformRoutineVersionOutputValues(values as RoutineVersionOutputShape, existing as RoutineVersionOutputShape, isCreate) as any;
    }
}

export async function validateRoutineVersionIOValues<IsInput extends boolean>(
    values: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isInput: IsInput,
    existing: IsInput extends true ? RoutineVersionInputShape : RoutineVersionOutputShape,
    isCreate: boolean,
) {
    if (isInput) {
        return validateFormValues(values as RoutineVersionInputShape, existing as RoutineVersionInputShape, isCreate, transformRoutineVersionInputValues, routineVersionInputValidation);
    } else {
        return validateFormValues(values as RoutineVersionOutputShape, existing as RoutineVersionOutputShape, isCreate, transformRoutineVersionOutputValues, routineVersionOutputValidation);
    }
}

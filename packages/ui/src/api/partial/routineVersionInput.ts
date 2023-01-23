import { RoutineVersionInput, RoutineVersionInputTranslation } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const routineVersionInputTranslationPartial: GqlPartial<RoutineVersionInputTranslation> = {
    __typename: 'RoutineVersionInputTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
}

export const routineVersionInputPartial: GqlPartial<RoutineVersionInput> = {
    __typename: 'RoutineVersionInput',
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: () => relPartial(require('./standardVersion').standardVersionPartial, 'list'),
        translations: () => relPartial(routineVersionInputTranslationPartial, 'full'),
    },
}
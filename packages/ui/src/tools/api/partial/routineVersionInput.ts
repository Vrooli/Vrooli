import { RoutineVersionInput, RoutineVersionInputTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const routineVersionInputTranslation: GqlPartial<RoutineVersionInputTranslation> = {
    __typename: 'RoutineVersionInputTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
    full: {},
    list: {},
}

export const routineVersionInput: GqlPartial<RoutineVersionInput> = {
    __typename: 'RoutineVersionInput',
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: async () => rel((await import('./standardVersion')).standardVersion, 'list'),
        translations: () => rel(routineVersionInputTranslation, 'full'),
    },
    full: {},
    list: {},
}
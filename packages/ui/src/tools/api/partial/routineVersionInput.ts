import { RoutineVersionInput, RoutineVersionInputTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const routineVersionInputTranslationPartial: GqlPartial<RoutineVersionInputTranslation> = {
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

export const routineVersionInputPartial: GqlPartial<RoutineVersionInput> = {
    __typename: 'RoutineVersionInput',
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'list'),
        translations: () => relPartial(routineVersionInputTranslationPartial, 'full'),
    },
    full: {},
    list: {},
}
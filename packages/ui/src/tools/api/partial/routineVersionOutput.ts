import { RoutineVersionOutput, RoutineVersionOutputTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const routineVersionOutputTranslation: GqlPartial<RoutineVersionOutputTranslation> = {
    __typename: 'RoutineVersionOutputTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
    full: {},
    list: {},
}

export const routineVersionOutput: GqlPartial<RoutineVersionOutput> = {
    __typename: 'RoutineVersionOutput',
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: async () => rel((await import('./standardVersion')).standardVersion, 'list'),
        translations: () => rel(routineVersionOutputTranslation, 'full'),
    },
    full: {},
    list: {},
}
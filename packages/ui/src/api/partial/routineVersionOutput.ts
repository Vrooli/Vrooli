import { RoutineVersionOutput, RoutineVersionOutputTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const routineVersionOutputTranslationPartial: GqlPartial<RoutineVersionOutputTranslation> = {
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

export const routineVersionOutputPartial: GqlPartial<RoutineVersionOutput> = {
    __typename: 'RoutineVersionOutput',
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: () => relPartial(require('./standardVersion').standardVersionPartial, 'list'),
        translations: () => relPartial(routineVersionOutputTranslationPartial, 'full'),
    },
    full: {},
    list: {},
}
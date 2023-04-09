import { NodeEnd } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const nodeEnd: GqlPartial<NodeEnd> = {
    __typename: 'NodeEnd',
    common: {
        id: true,
        wasSuccessful: true,
        suggestedNextRoutineVersions: async () => rel((await import('./routineVersion')).routineVersion, 'nav'),
    },
    full: {},
    list: {},
}
import { NodeEnd } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const nodeEndPartial: GqlPartial<NodeEnd> = {
    __typename: 'NodeEnd',
    common: {
        id: true,
        wasSuccessful: true,
        suggestedNextRoutineVersions: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
    },
    full: {},
    list: {},
}
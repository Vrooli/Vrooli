import { ProjectOrRoutine } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const projectOrRoutinePartial: GqlPartial<ProjectOrRoutine> = {
    __typename: 'ProjectOrRoutine' as any,
    full: {
        __define: {
            0: () => relPartial(require('./project').projectPartial, 'full'),
            1: () => relPartial(require('./routine').routinePartial, 'full'),
        },
        __union: {
            Project: 0,
            Routine: 1,
        }
    },
    list: {
        __define: {
            0: () => relPartial(require('./project').projectPartial, 'list'),
            1: () => relPartial(require('./routine').routinePartial, 'list'),
        },
        __union: {
            Project: 0,
            Routine: 1,
        }
    }
}
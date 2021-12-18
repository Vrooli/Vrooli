import { Routine, RoutineInput } from "schema/types";
import { creater, deleter, MODEL_TYPES, reporter, updater } from "./base";

export function RoutineModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Routine
    }

    return {
        ...obj,
        ...creater<RoutineInput, Routine>(obj),
        ...updater<RoutineInput, Routine>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}
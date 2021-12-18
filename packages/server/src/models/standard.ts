import { Standard, StandardInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, reporter, updater } from "./base";

export function StandardModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Standard
    }

    return {
        ...obj,
        ...findByIder<Standard>(obj),
        ...creater<StandardInput, Standard>(obj),
        ...updater<StandardInput, Standard>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}
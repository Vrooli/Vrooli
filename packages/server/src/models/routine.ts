import { Routine, RoutineInput } from "schema/types";
import { BaseState, creater, deleter, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

export function RoutineModel(prisma: any) {
    let obj: BaseState<Routine> = {
        prisma,
        model: MODEL_TYPES.Routine,
        format: formatter(),
    }

    return {
        ...obj,
        ...creater<RoutineInput, Routine>(obj),
        ...formatter(),
        ...updater<RoutineInput, Routine>(obj),
        ...deleter(obj),
        ...reporter()
    }
}
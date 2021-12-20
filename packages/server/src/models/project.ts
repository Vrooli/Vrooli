import { Project, ProjectInput } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

export function ProjectModel(prisma: any) {
    let obj: BaseState<Project> = {
        prisma,
        model: MODEL_TYPES.Project,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Project>(obj),
        ...formatter(),
        ...creater<ProjectInput, Project>(obj),
        ...updater<ProjectInput, Project>(obj),
        ...deleter(obj),
        ...reporter()
    }
}
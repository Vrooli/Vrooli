import { Project, ProjectInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, reporter, updater } from "./base";

export function ProjectModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Project
    }

    return {
        ...obj,
        ...findByIder<Project>(obj),
        ...creater<ProjectInput, Project>(obj),
        ...updater<ProjectInput, Project>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}
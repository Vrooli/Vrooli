import { Organization, OrganizationInput } from "schema/types";
import { creater, deleter, findByIder, MODEL_TYPES, reporter, updater } from "./base";

export function OrganizationModel(prisma: any) {
    let obj = {
        prisma,
        model: MODEL_TYPES.Organization
    }

    return {
        ...obj,
        ...findByIder<Organization>(obj),
        ...creater<OrganizationInput, Organization>(obj),
        ...updater<OrganizationInput, Organization>(obj),
        ...deleter(obj),
        ...reporter(obj)
    }
}
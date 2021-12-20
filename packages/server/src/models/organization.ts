import { Organization, OrganizationInput } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

export function OrganizationModel(prisma: any) {
    let obj: BaseState<Organization> = {
        prisma,
        model: MODEL_TYPES.Organization,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Organization>(obj),
        ...formatter(),
        ...creater<OrganizationInput, Organization>(obj),
        ...updater<OrganizationInput, Organization>(obj),
        ...deleter(obj),
        ...reporter()
    }
}
import { ProjectOrOrganization } from "@shared/consts";
import { GqlPartial } from "types";

export const projectOrOrganizationPartial: GqlPartial<ProjectOrOrganization> = {
    __typename: 'ProjectOrOrganization' as any,
    full: {
        __define: {
            0: [require('./project').projectPartial, 'full'],
            1: [require('./organization').organizationPartial, 'full'],
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    },
    list: {
        __define: {
            0: [require('./project').projectPartial, 'list'],
            1: [require('./organization').organizationPartial, 'list'],
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    }
}
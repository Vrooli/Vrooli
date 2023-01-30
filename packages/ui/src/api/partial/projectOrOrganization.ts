import { ProjectOrOrganization } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const projectOrOrganizationPartial: GqlPartial<ProjectOrOrganization> = {
    __typename: 'ProjectOrOrganization' as any,
    full: {
        __define: {
            0: () => relPartial(require('./project').projectPartial, 'full'),
            1: () => relPartial(require('./organization').organizationPartial, 'full'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    },
    list: {
        __define: {
            0: () => relPartial(require('./project').projectPartial, 'list'),
            1: () => relPartial(require('./organization').organizationPartial, 'list'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    }
}
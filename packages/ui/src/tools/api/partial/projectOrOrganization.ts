import { ProjectOrOrganization } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const projectOrOrganizationPartial: GqlPartial<ProjectOrOrganization> = {
    __typename: 'ProjectOrOrganization' as any,
    full: {
        __define: {
            0: async () => relPartial((await import('./project')).projectPartial, 'full'),
            1: async () => relPartial((await import('./organization')).organizationPartial, 'full'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    },
    list: {
        __define: {
            0: async () => relPartial((await import('./project')).projectPartial, 'list'),
            1: async () => relPartial((await import('./organization')).organizationPartial, 'list'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    }
}
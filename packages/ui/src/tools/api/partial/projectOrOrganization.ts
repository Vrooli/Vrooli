import { ProjectOrOrganization } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const projectOrOrganization: GqlPartial<ProjectOrOrganization> = {
    __typename: 'ProjectOrOrganization' as any,
    full: {
        __define: {
            0: async () => rel((await import('./project')).project, 'full'),
            1: async () => rel((await import('./organization')).organization, 'full'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    },
    list: {
        __define: {
            0: async () => rel((await import('./project')).project, 'list'),
            1: async () => rel((await import('./organization')).organization, 'list'),
        },
        __union: {
            Project: 0,
            Organization: 1,
        }
    }
}
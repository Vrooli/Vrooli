import { NodeLink } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const nodeLinkPartial: GqlPartial<NodeLink> = {
    __typename: 'NodeLink',
    common: {
        id: true,
        operation: true,
        whens: async () => relPartial((await import('./nodeLinkWhen')).nodeLinkWhenPartial, 'full', { omit: 'nodeLink' }),
    },
    full: {},
    list: {},
}
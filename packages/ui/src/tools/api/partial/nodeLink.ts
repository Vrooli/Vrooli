import { NodeLink } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const nodeLink: GqlPartial<NodeLink> = {
    __typename: 'NodeLink',
    common: {
        id: true,
        operation: true,
        whens: async () => rel((await import('./nodeLinkWhen')).nodeLinkWhen, 'full', { omit: 'nodeLink' }),
    },
    full: {},
    list: {},
}
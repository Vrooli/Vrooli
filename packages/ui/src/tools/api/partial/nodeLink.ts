import { NodeLink } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const nodeLinkPartial: GqlPartial<NodeLink> = {
    __typename: 'NodeLink',
    common: {
        id: true,
        operation: true,
        whens: () => relPartial(require('./nodeLinkWhen').nodeLinkWhenPartial, 'full', { omit: 'nodeLink' }),
    },
    full: {},
    list: {},
}
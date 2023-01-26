import { NodeLinkWhen, NodeLinkWhenTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const nodeLinkWhenTranslationPartial: GqlPartial<NodeLinkWhenTranslation> = {
    __typename: 'NodeLinkWhenTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const nodeLinkWhenPartial: GqlPartial<NodeLinkWhen> = {
    __typename: 'NodeLinkWhen',
    common: {
        id: true,
        condition: true,
        translations: () => relPartial(nodeLinkWhenTranslationPartial, 'full'),
    },
    full: {},
    list: {},
}
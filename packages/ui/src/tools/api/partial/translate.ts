import { Translate } from '@shared/consts';
import { GqlPartial } from '../types';

export const translate: GqlPartial<Translate> = {
    __typename: 'Translate',
    common: {
        fields: true,
        language: true,
    },
    full: {},
    list: {},
}
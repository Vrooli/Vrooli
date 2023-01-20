import { copyResultPartial } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const copyEndpoint = {
    copy: toMutation('copy', 'CopyInput', copyResultPartial, 'full'),
}
import { copyResultPartial } from 'api/partial';
import { toMutation } from 'api/utils';

export const copyEndpoint = {
    copy: toMutation('copy', 'CopyInput', copyResultPartial, 'full'),
}
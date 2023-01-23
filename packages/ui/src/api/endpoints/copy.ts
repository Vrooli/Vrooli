import { copyResultPartial } from '../partial';
import { toMutation } from '../utils';

export const copyEndpoint = {
    copy: toMutation('copy', 'CopyInput', copyResultPartial, 'full'),
}
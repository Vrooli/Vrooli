import { toQuery } from '../utils';

export const translateEndpoint = {
    translate: toQuery('translate', 'FindByIdInput', {} as any, 'full'),//translatePartial, 'full'),
}
import { toQuery } from 'graphql/utils';

export const translateEndpoint = {
    translate: toQuery('translate', 'FindByIdInput', [], `
        fields
        language
    `),
}
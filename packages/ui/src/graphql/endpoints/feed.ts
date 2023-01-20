import { developResultPartial, learnResultPartial, popularResultPartial, researchResultPartial } from 'graphql/partial';
import { toQuery } from 'graphql/utils';

export const feedEndpoint = {
    popular: toQuery('popular', 'PopularInput', popularResultPartial, 'list'),
    learn: toQuery('learn', null, learnResultPartial, 'list'),
    research: toQuery('research', null, researchResultPartial, 'list'),
    develop: toQuery('develop', null, developResultPartial, 'list'),
}
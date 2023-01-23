import { developResultPartial, learnResultPartial, popularResultPartial, researchResultPartial } from 'api/partial';
import { toQuery } from 'api/utils';

export const feedEndpoint = {
    popular: toQuery('popular', 'PopularInput', popularResultPartial, 'list'),
    learn: toQuery('learn', null, learnResultPartial, 'list'),
    research: toQuery('research', null, researchResultPartial, 'list'),
    develop: toQuery('develop', null, developResultPartial, 'list'),
}
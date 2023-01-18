import { ResourceList } from "@shared/consts";
import { uuid } from "@shared/uuid";

/**
 * Default ResourceList object 
 */
export const defaultResourceList: ResourceList = {
    __typename: 'ResourceList',
    id: uuid(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    resources: [],
    translations: [],
}
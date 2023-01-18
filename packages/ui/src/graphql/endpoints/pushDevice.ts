import { pushDeviceFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const pushDeviceEndpoint = {
    findMany: toQuery('pushDevices', 'PushDeviceSearchInput', toSearch(fullFields)),
    create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', fullFields[1]),
    update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', fullFields[1])
}
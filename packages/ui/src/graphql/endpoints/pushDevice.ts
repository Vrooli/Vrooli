import { pushDeviceFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const pushDeviceEndpoint = {
    findMany: toQuery('pushDevices', 'PushDeviceSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', [fullFields], `...fullFields`),
    update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', [fullFields], `...fullFields`)
}
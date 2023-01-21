import { pushDevicePartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const pushDeviceEndpoint = {
    findMany: toQuery('pushDevices', 'PushDeviceSearchInput', ...toSearch(pushDevicePartial)),
    create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', pushDevicePartial, 'full'),
    update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', pushDevicePartial, 'full')
}
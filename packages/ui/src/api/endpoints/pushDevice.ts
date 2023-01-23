import { pushDevicePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const pushDeviceEndpoint = {
    findMany: toQuery('pushDevices', 'PushDeviceSearchInput', ...toSearch(pushDevicePartial)),
    create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', pushDevicePartial, 'full'),
    update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', pushDevicePartial, 'full')
}
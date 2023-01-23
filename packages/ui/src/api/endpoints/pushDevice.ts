import { pushDevicePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const pushDeviceEndpoint = {
    findMany: toQuery('pushDevices', 'PushDeviceSearchInput', ...toSearch(pushDevicePartial)),
    create: toMutation('pushDeviceCreate', 'PushDeviceCreateInput', pushDevicePartial, 'full'),
    update: toMutation('pushDeviceUpdate', 'PushDeviceUpdateInput', pushDevicePartial, 'full')
}
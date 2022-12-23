import { apiVersionFields, noteVersionFields, organizationFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const copyEndpoint = {
    copy: toMutation('copy', 'CopyInput', [apiVersionFields, noteVersionFields, organizationFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields], `
        apiVersion {
            ...apiVersionFields
        }
        noteVersion {
            ...noteVersionFields
        }
        organization {
            ...organizationFields
        }
        projectVersion {
            ...projectVersionFields
        }
        routineVersion {
            ...routineVersionFields
        }
        smartContractVersion {
            ...smartContractVersionFields
        }
        standardVersion {
            ...standardVersionFields
        }
    `),
}
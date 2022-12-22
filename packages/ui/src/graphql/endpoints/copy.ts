import { apiVersionFields, noteVersionFields, organizationVersionFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const copyEndpoint = {
    copy: toGql('mutation', 'copy', 'CopyInput', [apiVersionFields, noteVersionFields, organizationVersionFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields], `
        apiVersion {
            ...apiVersionFields
        }
        noteVersion {
            ...noteVersionFields
        }
        organizationVersion {
            ...organizationVersionFields
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
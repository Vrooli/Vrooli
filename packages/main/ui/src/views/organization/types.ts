import { Organization } from ":local/consts";
import { UpsertProps, ViewProps } from "../types";

export interface OrganizationUpsertProps extends UpsertProps<Organization> { }
export interface OrganizationViewProps extends ViewProps<Organization> { }

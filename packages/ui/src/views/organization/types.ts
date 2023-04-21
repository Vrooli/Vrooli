import { Organization } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface OrganizationUpsertProps extends UpsertProps<Organization> { }
export interface OrganizationViewProps extends ViewProps<Organization> { }
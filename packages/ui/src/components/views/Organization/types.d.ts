import { Organization } from "types";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface OrganizationCreateProps extends CreateProps<Organization> {}
export interface OrganizationUpdateProps extends UpdateProps<Organization> {}
export interface OrganizationViewProps extends ViewProps<Organization> {}
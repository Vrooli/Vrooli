import { Organization } from "@local/shared";
import { FormProps } from "forms/types";
import { OrganizationShape } from "utils/shape/models/organization";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type OrganizationUpsertProps = UpsertProps<Organization>
export type OrganizationFormProps = FormProps<Organization, OrganizationShape>
export type OrganizationViewProps = ObjectViewProps<Organization>

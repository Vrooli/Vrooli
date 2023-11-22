import { Organization } from "@local/shared";
import { FormProps } from "forms/types";
import { OrganizationShape } from "utils/shape/models/organization";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type OrganizationUpsertPropsPage = CrudPropsPage;
type OrganizationUpsertPropsDialog = CrudPropsDialog<Organization>;
export type OrganizationUpsertProps = OrganizationUpsertPropsPage | OrganizationUpsertPropsDialog;
export type OrganizationFormProps = FormProps<Organization, OrganizationShape>
export type OrganizationViewProps = ObjectViewProps<Organization>

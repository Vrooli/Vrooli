import { Resource } from "@local/shared";
import { FormProps } from "forms/types";
import { ResourceShape } from "utils/shape/models/resource";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type ResourceUpsertPropsPage = CrudPropsPage & {
    isMutate: boolean;
};
type ResourceUpsertPropsDialog = CrudPropsDialog<Resource> & {
    isMutate: boolean;
};
export type ResourceUpsertProps = ResourceUpsertPropsPage | ResourceUpsertPropsDialog;
export type ResourceFormProps = FormProps<Resource, ResourceShape> & Pick<ResourceUpsertProps, "isMutate">;

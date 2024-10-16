import { Resource, ResourceShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps } from "../../../types";

type ResourceUpsertPropsPage = CrudPropsPage & {
    isMutate: boolean;
};
type ResourceUpsertPropsDialog = CrudPropsDialog<Resource> & {
    isMutate: boolean;
};
export type ResourceUpsertProps = ResourceUpsertPropsPage | ResourceUpsertPropsDialog;
export type ResourceFormProps = FormProps<Resource, ResourceShape> & Pick<ResourceUpsertProps, "isMutate">;

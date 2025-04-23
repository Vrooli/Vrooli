import { Resource, ResourceShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps } from "../../../types.js";

type ResourceUpsertPropsPage = CrudPropsPage & {
    isMutate: boolean;
};
type ResourceUpsertPropsDialog = CrudPropsDialog<Resource> & {
    isMutate: boolean;
};
type ResourceUpsertPropsPartial = CrudPropsPartial<Resource> & {
    isMutate: boolean;
}
export type ResourceUpsertProps = ResourceUpsertPropsPage | ResourceUpsertPropsDialog | ResourceUpsertPropsPartial;
export type ResourceFormProps = FormProps<Resource, ResourceShape> & Pick<ResourceUpsertProps, "isMutate">;

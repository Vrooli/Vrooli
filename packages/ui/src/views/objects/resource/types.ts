import { type Resource, type ResourceShape } from "@vrooli/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps } from "../../../types.js";

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

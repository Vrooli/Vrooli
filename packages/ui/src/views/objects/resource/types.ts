import { Resource } from "@local/shared";
import { FormProps } from "forms/types";
import { ResourceShape } from "utils/shape/models/resource";
import { CrudPropsDialog, CrudPropsPage } from "../types";
import { NewResourceShape } from "./ResourceUpsert/ResourceUpsert";

type ResourceUpsertPropsPage = CrudPropsPage & {
    isMutate: boolean;
};
type ResourceUpsertPropsDialog = Omit<CrudPropsDialog<Resource>, "overrideObject"> & {
    isMutate: boolean;
    overrideObject?: NewResourceShape;
};
export type ResourceUpsertProps = ResourceUpsertPropsPage | ResourceUpsertPropsDialog;
export type ResourceFormProps = FormProps<Resource, ResourceShape> & Pick<ResourceUpsertProps, "isMutate">;

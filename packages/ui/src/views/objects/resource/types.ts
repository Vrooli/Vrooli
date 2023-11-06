import { Resource } from "@local/shared";
import { FormProps } from "forms/types";
import { ResourceShape } from "utils/shape/models/resource";
import { UpsertProps } from "../types";
import { NewResourceShape } from "./ResourceUpsert/ResourceUpsert";

export type ResourceUpsertProps = Omit<UpsertProps<Resource>, "overrideObject"> & {
    isMutate: boolean;
    overrideObject?: NewResourceShape;
}
export type ResourceFormProps = FormProps<Resource, ResourceShape> & Pick<ResourceUpsertProps, "isMutate">;

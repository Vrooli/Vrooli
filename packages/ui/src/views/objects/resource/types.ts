import { Resource } from "@local/shared";
import { NewResourceShape } from "forms/ResourceForm/ResourceForm";
import { UpsertProps } from "../types";

export type ResourceUpsertProps = Omit<UpsertProps<Resource>, "overrideObject"> & {
    isMutate: boolean;
    overrideObject?: NewResourceShape;
}

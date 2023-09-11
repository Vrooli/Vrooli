import { Resource } from "@local/shared";
import { UpsertProps } from "../types";
import { NewResourceShape } from "./ResourceUpsert/ResourceUpsert";

export type ResourceUpsertProps = Omit<UpsertProps<Resource>, "overrideObject"> & {
    isMutate: boolean;
    overrideObject?: NewResourceShape;
}

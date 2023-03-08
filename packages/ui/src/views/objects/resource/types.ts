import { Resource } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface ResourceCreateProps extends CreateProps<Resource> {
    listId: string;
}
export interface ResourceUpdateProps extends UpdateProps<Resource> {
    listId: string;
}
export interface ResourceViewProps extends ViewProps<Resource> {}
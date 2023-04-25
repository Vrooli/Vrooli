import { ProjectVersion } from "@local/shared";
import { UpsertProps, ViewProps } from "../types";

export interface ProjectUpsertProps extends UpsertProps<ProjectVersion> { }
export interface ProjectViewProps extends ViewProps<ProjectVersion> { }
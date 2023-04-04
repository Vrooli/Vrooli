import { ProjectVersion } from "@shared/consts";
import { UpsertProps, ViewProps } from "../types";

export interface ProjectUpsertProps extends UpsertProps<ProjectVersion> { }
export interface ProjectViewProps extends ViewProps<ProjectVersion> { }
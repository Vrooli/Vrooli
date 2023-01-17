import { ProjectVersion } from "@shared/consts";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface ProjectCreateProps extends CreateProps<ProjectVersion> {}
export interface ProjectUpdateProps extends UpdateProps<ProjectVersion> {}
export interface ProjectViewProps extends ViewProps<ProjectVersion> {}
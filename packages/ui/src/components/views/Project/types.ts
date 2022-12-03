import { Project } from "types";
import { CreateProps, UpdateProps, ViewProps } from "../types";

export interface ProjectCreateProps extends CreateProps<Project> {}
export interface ProjectUpdateProps extends UpdateProps<Project> {}
export interface ProjectViewProps extends ViewProps<Project> {}
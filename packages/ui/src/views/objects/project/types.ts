import { ProjectVersion } from "@local/shared";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type ProjectUpsertProps = UpsertProps<ProjectVersion>
export type ProjectViewProps = ObjectViewProps<ProjectVersion>

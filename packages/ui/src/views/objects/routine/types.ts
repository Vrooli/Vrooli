import { RoutineVersion } from "@local/shared";
import { FormProps } from "forms/types";
import { RoutineVersionShape } from "utils/shape/models/routineVersion";
import { ObjectViewProps, ViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

/** RoutineVersion with fields required for the build view */
export type BuildRoutineVersion = Pick<RoutineVersion, "id" | "nodes" | "nodeLinks" | "translations">

export type BuildViewProps = ViewProps & {
    handleCancel: () => unknown;
    handleSubmit: (updatedRoutineVersion: BuildRoutineVersion) => unknown;
    isEditing: boolean;
    loading: boolean;
    routineVersion: BuildRoutineVersion;
    translationData: {
        language: string;
        setLanguage: (language: string) => unknown;
        handleAddLanguage: (language: string) => unknown;
        handleDeleteLanguage: (language: string) => unknown;
        languages: string[];
    };
};

type RoutineUpsertPropsPage = CrudPropsPage & {
    isSubroutine?: boolean;
}
type RoutineUpsertPropsDialog = CrudPropsDialog<RoutineVersion> & {
    isSubroutine?: boolean;
};
export type RoutineUpsertProps = RoutineUpsertPropsPage | RoutineUpsertPropsDialog;
export type RoutineFormProps = FormProps<RoutineVersion, RoutineVersionShape> & Pick<RoutineUpsertProps, "isSubroutine"> & {
    versions: string[];
}
export type RoutineViewProps = ObjectViewProps<RoutineVersion>

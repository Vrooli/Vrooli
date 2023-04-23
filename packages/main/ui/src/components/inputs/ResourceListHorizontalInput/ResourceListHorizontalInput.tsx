import { ResourceList } from "@local/consts";
import { useField } from "formik";
import { useCallback } from "react";
import { ResourceListHorizontal } from "../../lists/resource";
import { ResourceListHorizontalInputProps } from "../types";

export const ResourceListHorizontalInput = ({
    disabled = false,
    isCreate,
    isLoading = false,
    zIndex,
}: ResourceListHorizontalInputProps) => {
    const [field, , helpers] = useField("resourceList");

    const handleUpdate = useCallback((newList: ResourceList) => {
        helpers.setValue(newList);
    }, [helpers]);

    return (
        <ResourceListHorizontal
            list={field.value}
            canUpdate={!disabled}
            handleUpdate={handleUpdate}
            loading={isLoading}
            mutate={!isCreate}
            zIndex={zIndex}
        />
    );
};

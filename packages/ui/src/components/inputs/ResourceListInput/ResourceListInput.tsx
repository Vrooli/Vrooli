import { ResourceList as ResourceListType } from "@local/shared";
import { ResourceList } from "components/lists/resource";
import { useField } from "formik";
import { useCallback } from "react";
import { ResourceListInputProps } from "../types";

export const ResourceListInput = ({
    disabled = false,
    horizontal,
    isCreate,
    isLoading = false,
    parent,
}: ResourceListInputProps) => {
    const [field, , helpers] = useField("resourceList");

    const handleUpdate = useCallback((newList: ResourceListType) => {
        helpers.setValue(newList);
    }, [helpers]);

    return (
        <ResourceList
            horizontal={horizontal}
            list={field.value}
            canUpdate={!disabled}
            handleUpdate={handleUpdate}
            loading={isLoading}
            mutate={!isCreate}
            parent={parent}
        />
    );
};

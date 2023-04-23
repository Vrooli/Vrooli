import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { useCallback } from "react";
import { ResourceListHorizontal } from "../../lists/resource";
export const ResourceListHorizontalInput = ({ disabled = false, isCreate, isLoading = false, zIndex, }) => {
    const [field, , helpers] = useField("resourceList");
    const handleUpdate = useCallback((newList) => {
        helpers.setValue(newList);
    }, [helpers]);
    return (_jsx(ResourceListHorizontal, { list: field.value, canUpdate: !disabled, handleUpdate: handleUpdate, loading: isLoading, mutate: !isCreate, zIndex: zIndex }));
};
//# sourceMappingURL=ResourceListHorizontalInput.js.map
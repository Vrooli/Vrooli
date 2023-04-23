import { jsx as _jsx } from "react/jsx-runtime";
import { exists } from "@local/utils";
import { useField } from "formik";
import { useCallback } from "react";
import { TagSelectorBase } from "../TagSelectorBase/TagSelectorBase";
export const TagSelector = ({ disabled, name, placeholder = "Enter tags, followed by commas...", }) => {
    const [field, , helpers] = useField(name);
    const handleTagsUpdate = useCallback((tags) => {
        exists(helpers) && helpers.setValue(tags);
    }, [helpers]);
    return (_jsx(TagSelectorBase, { disabled: disabled, handleTagsUpdate: handleTagsUpdate, placeholder: placeholder, tags: field.value ?? [] }));
};
//# sourceMappingURL=TagSelector.js.map
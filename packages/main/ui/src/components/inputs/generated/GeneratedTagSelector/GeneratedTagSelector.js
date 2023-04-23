import { jsx as _jsx } from "react/jsx-runtime";
import { useField } from "formik";
import { TagSelectorBase } from "../../TagSelectorBase/TagSelectorBase";
export const GeneratedTagSelector = ({ disabled, fieldData, index, }) => {
    console.log("rendering tag selector");
    const [field, , helpers] = useField(fieldData.fieldName);
    return (_jsx(TagSelectorBase, { disabled: disabled, handleTagsUpdate: helpers.setValue, tags: field.value }));
};
//# sourceMappingURL=GeneratedTagSelector.js.map
import { Tag, TagSelectorFormInput, TagShape } from "@local/shared";
import { TagSelectorBase } from "components/inputs/TagSelector/TagSelector";
import { useField } from "formik";
import { FormInputProps } from "../types";

export function FormInputTagSelector({
    disabled,
    fieldData,
    index,
}: FormInputProps<TagSelectorFormInput>) {
    const [field, , helpers] = useField<(Tag | TagShape)[]>(fieldData.fieldName);

    return (
        <TagSelectorBase
            disabled={disabled}
            handleTagsUpdate={helpers.setValue}
            tags={field.value}
        />
    );
}

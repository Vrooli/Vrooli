import { type Tag, type TagSelectorFormInput, type TagShape } from "@local/shared";
import { useField } from "formik";
import { TagSelectorBase } from "../TagSelector/TagSelector.js";
import { type FormInputProps } from "./types.js";

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

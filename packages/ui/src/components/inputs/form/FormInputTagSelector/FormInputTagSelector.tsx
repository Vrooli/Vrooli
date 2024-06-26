import { Tag } from "@local/shared";
import { TagSelectorBase } from "components/inputs/TagSelector/TagSelector";
import { useField } from "formik";
import { TagSelectorFormInput } from "forms/types";
import { TagShape } from "utils/shape/models/tag";
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

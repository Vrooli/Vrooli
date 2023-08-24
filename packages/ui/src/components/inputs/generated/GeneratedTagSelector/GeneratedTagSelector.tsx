import { Tag } from "@local/shared";
import { TagSelectorBase } from "components/inputs/TagSelectorBase/TagSelectorBase";
import { useField } from "formik";
import { TagShape } from "utils/shape/models/tag";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTagSelector = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    const [field, , helpers] = useField<(Tag | TagShape)[]>(fieldData.fieldName);

    return (
        <TagSelectorBase
            disabled={disabled}
            handleTagsUpdate={helpers.setValue}
            tags={field.value}
        />
    );
};

import { Tag } from ":/consts";
import { useField } from "formik";
import { TagShape } from "../../../../utils/shape/models/tag";
import { TagSelectorBase } from "../../TagSelectorBase/TagSelectorBase";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTagSelector = ({
    disabled,
    fieldData,
    index,
}: GeneratedInputComponentProps) => {
    console.log("rendering tag selector");

    const [field, , helpers] = useField<(Tag | TagShape)[]>(fieldData.fieldName);

    return (
        <TagSelectorBase
            disabled={disabled}
            handleTagsUpdate={helpers.setValue}
            tags={field.value}
        />
    );
};

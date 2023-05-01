import { exists, Tag } from "@local/shared";
import { useField } from "formik";
import { useCallback } from "react";
import { TagShape } from "utils/shape/models/tag";
import { TagSelectorBase } from "../TagSelectorBase/TagSelectorBase";
import { TagSelectorProps } from "../types";

export const TagSelector = ({
    disabled,
    name,
    placeholder = "Enter tags, followed by commas...",
    zIndex,
}: TagSelectorProps) => {
    const [field, , helpers] = useField<(TagShape | Tag)[] | undefined>(name);

    const handleTagsUpdate = useCallback((tags: (TagShape | Tag)[]) => {
        exists(helpers) && helpers.setValue(tags);
    }, [helpers]);

    return (
        <TagSelectorBase
            disabled={disabled}
            handleTagsUpdate={handleTagsUpdate}
            placeholder={placeholder}
            tags={field.value ?? []}
            zIndex={zIndex}
        />
    );
};

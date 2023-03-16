import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { useCallback, useMemo } from "react";
import { TagShape } from "utils/shape/models/tag";
import { GeneratedInputComponentProps } from "../types";

export const GeneratedTagSelector = ({
    disabled,
    fieldData,
    formik,
    index,
    session,
}: GeneratedInputComponentProps) => {
    console.log('rendering tag selector');
    const tags = useMemo(() => formik.values[fieldData.fieldName] as TagShape[], [formik.values, fieldData.fieldName]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { formik.setFieldValue(fieldData.fieldName, updatedList) }, [formik, fieldData.fieldName]);

    return (
        <TagSelector
            disabled={disabled}
            handleTagsUpdate={handleTagsUpdate}
            session={session}
            tags={tags}
        />
    );
}
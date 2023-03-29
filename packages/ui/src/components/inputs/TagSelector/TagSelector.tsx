import { Tag } from '@shared/consts';
import { exists } from '@shared/utils';
import { useField } from 'formik';
import { useCallback } from 'react';
import { TagShape } from 'utils/shape/models/tag';
import { TagSelectorBase } from '../TagSelectorBase/TagSelectorBase';
import { TagSelectorProps } from '../types';

export const TagSelector = ({
    disabled,
    placeholder = 'Enter tags, followed by commas...',
}: TagSelectorProps) => {
    const [versionField, , versionHelpers] = useField<(TagShape | Tag)[] | undefined>('tags');
    const [rootField, , rootHelpers] = useField<(TagShape | Tag)[] | undefined>('root.tags');

    const handleTagsUpdate = useCallback((tags: (TagShape | Tag)[]) => {
        exists(versionHelpers) && versionHelpers.setValue(tags);
        exists(rootHelpers) && rootHelpers.setValue(tags);
    }, [rootHelpers, versionHelpers]);

    return (
        <TagSelectorBase
            disabled={disabled}
            handleTagsUpdate={handleTagsUpdate}
            placeholder={placeholder}
            tags={versionField.value ?? rootField.value ?? []}
        />
    )
}
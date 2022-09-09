/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
 */
import { EditableTextCollapseProps } from '../types';
import { ContentCollapse } from 'components';
import Markdown from 'markdown-to-jsx';
import { useMemo } from 'react';

export function EditableTextCollapse({
    helpText,
    isEditing,
    isOpen,
    onOpenChange,
    showOnNoText,
    title,
    text,
}: EditableTextCollapseProps) {
    if ((!text || text.trim().length === 0) && !showOnNoText) return null;

    const content = useMemo(() => {

    }, [isEditing, text]);
    
    //TODO
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
        >
            {text ? <Markdown>{text}</Markdown> : null}
        </ContentCollapse>
    )
}
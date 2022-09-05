/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
 */
import { EditableTextCollapseProps } from '../types';
import { ContentCollapse } from 'components';
import Markdown from 'markdown-to-jsx';

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
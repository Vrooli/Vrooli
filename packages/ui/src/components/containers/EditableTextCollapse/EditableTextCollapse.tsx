/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
 */
import { EditableTextCollapseProps } from '../types';
import { ContentCollapse, MarkdownInput } from 'components';
import Markdown from 'markdown-to-jsx';
import { TextField } from '@mui/material';

export function EditableTextCollapse({
    helpText,
    isEditing,
    isOpen,
    onOpenChange,
    propsTextField,
    propsMarkdownInput,
    showOnNoText,
    title,
    text,
}: EditableTextCollapseProps) {
    if (!isEditing && (!text || text.trim().length === 0) && !showOnNoText) return null;

    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
        >
            {
                isEditing ? 
                    propsMarkdownInput ?
                        <MarkdownInput {...propsMarkdownInput} /> :
                        <TextField {...(propsTextField ?? {})} /> :
                    text ? <Markdown>{text}</Markdown> : null
            }
        </ContentCollapse>
    )
}
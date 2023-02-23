/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
 */
import { EditableTextCollapseProps } from '../types';
import { ContentCollapse, MarkdownInput } from 'components';
import Markdown from 'markdown-to-jsx';
import { TextField, useTheme } from '@mui/material';
import { linkColors } from 'styles';

export function EditableTextCollapse({
    helpText,
    isEditing,
    isOpen,
    onOpenChange,
    propsTextField,
    propsMarkdownInput,
    session,
    showOnNoText,
    title,
    text,
}: EditableTextCollapseProps) {
    const { palette } = useTheme();

    if (!isEditing && (!text || text.trim().length === 0) && !showOnNoText) return null;
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            session={session}
            title={title}
            sxs={{
                root: { ...linkColors(palette) },
            }}
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
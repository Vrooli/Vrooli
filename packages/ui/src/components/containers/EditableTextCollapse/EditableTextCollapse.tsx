/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
 */
import { TextField, useTheme } from '@mui/material';
import { MarkdownInput } from 'components/inputs/MarkdownInput/MarkdownInput';
import Markdown from 'markdown-to-jsx';
import { linkColors } from 'styles';
import { ContentCollapse } from '../ContentCollapse/ContentCollapse';
import { EditableTextCollapseProps } from '../types';

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
    const { palette } = useTheme();

    if (!isEditing && (!text || text.trim().length === 0) && !showOnNoText) return null;
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
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
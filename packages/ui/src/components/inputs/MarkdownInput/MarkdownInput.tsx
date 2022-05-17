/**
 * TextField for entering (and previewing) markdown.
 */
import { useCallback, useEffect, useState } from 'react';
import { Box, Button, IconButton, Popover, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import {
    FormatListBulleted as BulletListIcon,
    FormatListNumbered as NumberListIcon,
    FormatBold as BoldIcon,
    FormatItalic as ItalicIcon,
    InsertLink as LinkIcon,
    List as ListIcon,
    StrikethroughS as StrikethroughIcon,
    Title as HeaderIcon,
    Visibility as PreviewOnIcon,
    VisibilityOff as PreviewOffIcon,
} from '@mui/icons-material';
import { MarkdownInputProps } from '../types';
import { Pubs } from 'utils';
import Markdown from 'markdown-to-jsx';
import { noSelect } from 'styles';

enum Headers {
    H1 = 'h1',
    H2 = 'h2',
    H3 = 'h3',
    H4 = 'h4',
    H5 = 'h5',
    H6 = 'h6',
}
const headerMarkdowns = {
    [Headers.H1]: '# ',
    [Headers.H2]: '## ',
    [Headers.H3]: '### ',
    [Headers.H4]: '#### ',
    [Headers.H5]: '##### ',
    [Headers.H6]: '###### ',
}

// TODO - changing textarea programmatically breaks undo/redo functionality
export const MarkdownInput = ({
    id,
    disabled = false,
    error = false,
    helperText,
    minRows = 4,
    onChange,
    placeholder = '',
    value,
}: MarkdownInputProps) => {
    const { palette } = useTheme();

    const [isPreviewOn, setIsPreviewOn] = useState(false);

    const [headerAnchorEl, setHeaderAnchorEl] = useState(null);
    const openHeaderSelect = (event) => { setHeaderAnchorEl(event.currentTarget) };
    const closeHeader = () => { setHeaderAnchorEl(null) };
    const headerSelectOpen = Boolean(headerAnchorEl);

    const [listAnchorEl, setListAnchorEl] = useState(null);
    const openListSelect = (event) => { setListAnchorEl(event.currentTarget) };
    const closeList = () => { setListAnchorEl(null) };
    const listSelectOpen = Boolean(listAnchorEl);

    // Listen for text input changes
    useEffect(() => {
        const handleKeyDown = (e: any) => {
            // On enter key, check if bullet or number should be added to new line
            if (e.key === 'Enter') {
                // TODO
            }
        }
        const textarea = document.getElementById(`markdown-input-${id}`);
        if (!textarea) return;
        // Add listener for key press
        textarea.addEventListener('keydown', handleKeyDown);
        // Remove listener on unmount
        return () => textarea.removeEventListener('keydown', handleKeyDown);
    }, [id]);

    const insertHeader = useCallback((header: Headers) => {
        // Find the textarea element
        const textarea = document.getElementById(`markdown-input-${id}`) as HTMLTextAreaElement;
        if (!textarea) return;
        // Find the current cursor position. Ignore any selection
        const startPosition = textarea.selectionStart;
        // Find the start of the line which the select starts on
        let startLine = textarea.value.substring(0, startPosition).lastIndexOf('\n') + 1;
        // If not found, add to the beginning
        if (startLine === -1) startLine = 0;
        // Insert the header at the start of the line
        textarea.value = textarea.value.substring(0, startLine) + headerMarkdowns[header] + textarea.value.substring(startLine);
        onChange(textarea.value);
        closeHeader();
    }, [id, onChange]);

    /**
     * Pads selection with the given substring
     * @param padStart The substring to add before the selection
     * @param padEnd The substring to add after the selection
     */
    const padSelection = useCallback((padStart: string, padEnd: string) => {
        // Find the textarea element
        const textarea = document.getElementById(`markdown-input-${id}`) as HTMLTextAreaElement;
        if (!textarea) return;
        // Find the current selection
        const startPosition = textarea.selectionStart;
        const endPosition = textarea.selectionEnd;
        // If no selection, return
        if (startPosition === endPosition) {
            PubSub.publish(Pubs.Snack, { message: 'No text selected', severity: 'Error' });
            return;
        }
        // Insert ~~ before the selection, and ~~ after the selection
        textarea.value = textarea.value.substring(0, startPosition) + padStart + textarea.value.substring(startPosition, endPosition) + padEnd + textarea.value.substring(endPosition);
        onChange(textarea.value);
    }, [id, onChange]);

    const strikethrough = useCallback(() => { padSelection('~~', '~~') }, [padSelection]);
    const bold = useCallback(() => { padSelection('**', '**') }, [padSelection]);
    const italic = useCallback(() => { padSelection('*', '*') }, [padSelection]);

    const insertLink = useCallback(() => {
        // Find the textarea element
        const textarea = document.getElementById(`markdown-input-${id}`) as HTMLTextAreaElement;
        if (!textarea) return;
        // Find the current selection
        const startPosition = textarea.selectionStart;
        const endPosition = textarea.selectionEnd;
        // If no selection, insert [link](url) at the cursor
        if (startPosition === endPosition) {
            textarea.value = textarea.value.substring(0, startPosition) + '[display text](url)' + textarea.value.substring(endPosition);
            onChange(textarea.value);
            return;
        }
        // Otherwise, call padSelection
        padSelection('[', '](url)');
    }, [id, onChange, padSelection]);

    const insertBulletList = useCallback(() => {
        // Find the textarea element
        const textarea = document.getElementById(`markdown-input-${id}`) as HTMLTextAreaElement;
        if (!textarea) return;
        const startPosition = textarea.selectionStart;
        const endPosition = textarea.selectionEnd;
        // Find the start of the line which the select starts on
        let startLine = textarea.value.substring(0, startPosition).lastIndexOf('\n') + 1;
        // If not found, set to beginning
        if (startLine === -1) startLine = 0;
        // Create new textarea value
        let newText = textarea.value.substring(0, startLine) + '* ';
        for (let i = startLine; i <= endPosition; i++) {
            const currChar = textarea.value.charAt(i);
            if (currChar === '\n') {
                newText += '\n* ';
            } else {
                newText += currChar;
            }
        }
        newText += textarea.value.substring(endPosition);
        // Set the new textarea value
        textarea.value = newText;
        onChange(textarea.value);
        closeList();
    }, [id, onChange]);

    const insertNumberList = useCallback(() => {
        // Find the textarea element
        const textarea = document.getElementById(`markdown-input-${id}`) as HTMLTextAreaElement;
        if (!textarea) return;
        const startPosition = textarea.selectionStart;
        const endPosition = textarea.selectionEnd;
        // Find the start of the line which the select starts on
        let startLine = textarea.value.substring(0, startPosition).lastIndexOf('\n') + 1;
        // If not found, set to beginning
        if (startLine === -1) startLine = 0;
        // Create new textarea value
        let newText = textarea.value.substring(0, startLine) + '1. ';
        let lineNumber = 2;
        for (let i = startLine; i <= endPosition; i++) {
            const currChar = textarea.value.charAt(i);
            if (currChar === '\n') {
                newText += '\n' + lineNumber + '. ';
                lineNumber++;
            } else {
                newText += currChar;
            }
        }
        newText += textarea.value.substring(endPosition);
        // Set the new textarea value
        textarea.value = newText;
        onChange(textarea.value);
        closeList();
    }, [id, onChange]);

    const togglePreview = useCallback(() => { setIsPreviewOn(on => !on) }, []);

    return (
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for inserting markdown and previewing */}
            <Box sx={{
                display: 'flex',
                width: '100%',
                padding: '0.5rem',
                borderBottom: '1px solid #e0e0e0',
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: '0.5rem 0.5rem 0 0',
            }}>
                {/* To the left is a stack for inserting titles, italics/bold, lists, and links */}
                <Stack direction="row" spacing={1} sx={{ marginRight: 'auto' }}>
                    {/* Insert header selector */}
                    <Tooltip title="Insert header" placement="top">
                        <IconButton aria-describedby={`markdown-input-header-popover-${id}`} size="small" onClick={openHeaderSelect}>
                            <HeaderIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    <Popover
                        id={`markdown-input-header-popover-${id}`}
                        open={headerSelectOpen}
                        anchorEl={headerAnchorEl}
                        onClose={closeHeader}
                        anchorOrigin={{
                            vertical: 'bottom',
                            horizontal: 'center',
                        }}
                    >
                        {/* When opened, button row of 1-6, for each size */}
                        <Stack direction="row" spacing={0} sx={{
                            background: palette.primary.light,
                            color: palette.primary.contrastText,
                        }}>
                            {
                                [Headers.H1, Headers.H2, Headers.H3].map((size) => (
                                    <Button
                                        key={size}
                                        color="primary"
                                        size="small"
                                        onClick={() => insertHeader(size as Headers)}
                                        sx={{
                                            minHeight: '48px',
                                        }}
                                    >
                                        {size}
                                    </Button>
                                ))
                            }
                        </Stack>
                    </Popover>
                    {/* Button for bold */}
                    <Tooltip title="Bold" placement="top">
                        <IconButton size="small" onClick={bold}>
                            <BoldIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Button for italic */}
                    <Tooltip title="Italic" placement="top">
                        <IconButton size="small" onClick={italic}>
                            <ItalicIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Button for strikethrough */}
                    <Tooltip title="Strikethrough" placement="top">
                        <IconButton size="small" onClick={strikethrough}>
                            <StrikethroughIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Insert bulleted or numbered list selector */}
                    <Tooltip title="Insert list" placement="top">
                        <IconButton aria-describedby={`markdown-input-list-popover-${id}`} size="small" onClick={openListSelect}>
                            <ListIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    <Popover
                        id={`markdown-input-list-popover-${id}`}
                        open={listSelectOpen}
                        anchorEl={listAnchorEl}
                        onClose={closeList}
                        anchorOrigin={{
                            vertical: 'bottom',
                            horizontal: 'center',
                        }}
                    >
                        <Stack direction="row" spacing={0} sx={{
                            background: palette.primary.light,
                            color: palette.primary.contrastText,
                        }}>
                            <Tooltip title="Bulleted list" placement="top">
                                <Button
                                    color="primary"
                                    size="small"
                                    onClick={insertBulletList}
                                    sx={{ minHeight: '48px', }}
                                    startIcon={<BulletListIcon sx={{
                                        marginLeft: 1,
                                        fill: palette.primary.contrastText,
                                        transform: 'scale(1.5)',
                                    }} />}
                                />
                            </Tooltip>
                            <Tooltip title="Numbered list" placement="top">
                                <Button
                                    color="primary"
                                    size="small"
                                    onClick={insertNumberList}
                                    sx={{ minHeight: '48px', }}
                                    startIcon={<NumberListIcon sx={{
                                        marginLeft: 1,
                                        fill: palette.primary.contrastText,
                                        transform: 'scale(1.5)',
                                    }} />}
                                />
                            </Tooltip>
                        </Stack>
                    </Popover>
                    {/* Button for inserting link */}
                    <Tooltip title="Insert link" placement="top">
                        <IconButton size="small" onClick={insertLink}>
                            <LinkIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                {/* To the right is a button for previewing the markdown */}
                <Stack direction="row" spacing={1}>
                    {/* Preview button */}
                    <Tooltip title={isPreviewOn ? 'Preview mode' : 'Edit mode'} placement="top">
                        <IconButton size="small" onClick={togglePreview}>
                            {
                                isPreviewOn ?
                                    <PreviewOffIcon sx={{ fill: palette.primary.contrastText }} /> :
                                    <PreviewOnIcon sx={{ fill: palette.primary.contrastText }} />
                            }
                        </IconButton>
                    </Tooltip>
                </Stack>
            </Box>
            {/* TextField for entering markdown, or markdown display if previewing */}
            {
                isPreviewOn ?
                    (
                        <Box sx={{
                            border: `1px solid ${error ? 'red' : 'black'}`,
                            borderRadius: '0 0 0.5rem 0.5rem',
                            borderTop: 'none',
                            ...noSelect,
                        }}>
                            <Markdown>{value}</Markdown>
                        </Box>
                    ) :
                    (
                        <textarea
                            id={`markdown-input-${id}`}
                            disabled={disabled}
                            placeholder={placeholder}
                            rows={minRows}
                            value={value}
                            onChange={(e) => { onChange(e.target.value) }}
                            style={{
                                minWidth: '-webkit-fill-available',
                                maxWidth: '-webkit-fill-available',
                                minHeight: '50px',
                                maxHeight: '800px',
                                background: 'transparent',
                                borderColor: error ? 'red' : 'unset',
                                borderRadius: '0 0 0.5rem 0.5rem',
                                borderTop: 'none',
                                fontFamily: 'inherit',
                                fontSize: 'inherit',
                                color: palette.text.primary
                            }}
                        />
                    )
            }
            {/* Helper text label */}
            {
                helperText &&
                <Typography variant="body1" sx={{ color: 'red' }}>
                    {helperText}
                </Typography>
            }
        </Stack>
    );
}
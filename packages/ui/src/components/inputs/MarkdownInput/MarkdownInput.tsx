/**
 * TextField for entering (and previewing) markdown.
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
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
} from '@mui/icons-material';
import { MarkdownInputProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { noSelect } from 'styles';
import { PubSub } from 'utils';
import { InvisibleIcon, VisibleIcon } from '@shared/icons';
import AwesomeDebouncePromise from 'awesome-debounce-promise';

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

/**
 * Determines start index of the current line.
 * @param text Text to search.
 * @param selectionStart Index of cursor or selection start.
 * @returns Index of start of current line.
 */
const getLineStart = (text: string, selectionStart: number) => {
    if (selectionStart < 0 || selectionStart > text.length) return 0;
    return text.substring(0, selectionStart).lastIndexOf('\n') + 1;
}

/**
 * Determines end index of the current line.
 * @param text Text to search.
 * @param selectionStart Index of cursor or selection start.
 * @returns Index of end of current line.
 */
const getLineEnd = (text: string, selectionStart: number) => {
    if (selectionStart < 0 || selectionStart > text.length) return text.length;
    return text.substring(selectionStart).indexOf('\n') + selectionStart;
}


/**
 * Determines line the cursor is on
 * @param text The entire text
 * @param selectionStart The index of the cursor, or start of highlighted text
 * @returns The line the cursor is on (or null), as well as its start and end index
 */
const getLineOnStart = (text: string, selectionStart: number): [string, number, number] => {
    const start = getLineStart(text, selectionStart);
    const end = getLineEnd(text, selectionStart);
    const line = text.substring(start, end);
    return [line, start, end];
}

/**
 * Determines all lines the cursor or highlighted text is on
 * @param text The entire text
 * @param selectionStart The index of the cursor, or start of highlighted text
 * @param selectionEnd The index of the end of highlighted text
 * @returns The lines the cursor is on (or null), as well as their start and end index
 */
const getLinesOnSelection = (text: string, selectionStart: number, selectionEnd: number): [string[], number, number] => {
    const start = getLineStart(text, selectionStart);
    const end = getLineEnd(text, selectionEnd);
    const lines = text.substring(start, end).split('\n');
    return [lines, start, end]
}

/**
 * Replaces selected text with new text.
 * @param text Text to replace in.
 * @param newText Text to replace with.
 * @param selectionStart Index of cursor or selection start.
 * @param selectionEnd Index of cursor or selection end.
 * @returns New text with selected text replaced.
 */
const replaceText = (text: string, newText: string, selectionStart: number, selectionEnd: number): string => {
    return text.substring(0, selectionStart) + newText + text.substring(selectionEnd);
}

/**
 * Uses element ID to get selectionStart, selectionEnd, and element.
 * @param id The ID of the element to get the selection of
 * @returns Object containing selectionStart, selectionEnd, and element
 */
const getSelection = (id: string): { selectionStart: number, selectionEnd: number, textArea: HTMLTextAreaElement } => {
    const textArea = document.getElementById(id);
    if (!textArea || !(textArea instanceof HTMLTextAreaElement)) throw new Error(`Element not found: ${id}`);
    return { selectionStart: textArea.selectionStart, selectionEnd: textArea.selectionEnd, textArea };
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

    // Internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Debounce text change
    const onChangeDebounced = useMemo(() => AwesomeDebouncePromise(
        onChange,
        100,
    ), [onChange]);
    useEffect(() => setInternalValue(value), [value]);
    const handleChange = useCallback((newValue: string) => {
        // Update state
        setInternalValue(newValue);
        // Debounce onChange
        onChangeDebounced(newValue);
    }, [onChangeDebounced]);

    const [isPreviewOn, setIsPreviewOn] = useState(false);

    const [headerAnchorEl, setHeaderAnchorEl] = useState<HTMLElement | null>(null);
    const openHeaderSelect = (event) => { setHeaderAnchorEl(event.currentTarget) };
    const closeHeader = () => { setHeaderAnchorEl(null) };
    const headerSelectOpen = Boolean(headerAnchorEl);

    const [listAnchorEl, setListAnchorEl] = useState<HTMLElement | null>(null);
    const openListSelect = (event) => { setListAnchorEl(event.currentTarget) };
    const closeList = () => { setListAnchorEl(null) };
    const listSelectOpen = Boolean(listAnchorEl);

    // Listen for text input changes
    useEffect(() => {
        const handleKeyDown = (e: any) => {
            // On enter key, check if bullet or number should be added to new line
            if (e.key === 'Enter') {
                // // selectionStart is used to determine the line the cursor is on
                // const { selectionStart, selectionEnd, value } = e.target;
                // const trimmedLine = getLineOnStart(value, selectionStart)?.trimStart() ?? '';
                // // Is a bullet if line starts with an asterisk or dash, followed by a space
                // const isBullet = trimmedLine.startsWith('* ') || trimmedLine.trim()?.startsWith('- ');
                // // Is a number if line starts with a number, followed by a period and a space
                // const isNumber = /^\d+\.\s/.test(trimmedLine);
                // console.log('enter key', trimmedLine, selectionStart, isBullet, isNumber);
                // if (isBullet || isNumber) {
                //     console.log('adding bullet or number');
                //     //TODO
                // }
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
        // Find the current selection
        const { selectionStart, textArea } = getSelection(`markdown-input-${id}`);
        // Find the start of the line which the select starts on
        const startLine = getLineStart(textArea.value, selectionStart);
        // Insert the header at the start of the line
        textArea.value = textArea.value.substring(0, startLine) + headerMarkdowns[header] + textArea.value.substring(startLine);
        handleChange(textArea.value);
        closeHeader();
    }, [handleChange, id]);

    /**
     * Pads selection with the given substring
     * @param padStart The substring to add before the selection
     * @param padEnd The substring to add after the selection
     */
    const padSelection = useCallback((padStart: string, padEnd: string) => {
        // Find the current selection
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${id}`);
        // If no selection, return
        if (selectionStart === selectionEnd) {
            PubSub.get().publishSnack({ message: 'No text selected', severity: 'Error' });
            return;
        }
        // Insert ~~ before the selection, and ~~ after the selection
        textArea.value = textArea.value.substring(0, selectionStart) + padStart + textArea.value.substring(selectionStart, selectionEnd) + padEnd + textArea.value.substring(selectionEnd);
        handleChange(textArea.value);
    }, [handleChange, id]);

    const strikethrough = useCallback(() => { padSelection('~~', '~~') }, [padSelection]);
    const bold = useCallback(() => { padSelection('**', '**') }, [padSelection]);
    const italic = useCallback(() => { padSelection('*', '*') }, [padSelection]);

    const insertLink = useCallback(() => {
        // Find the current selection
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${id}`);
        // If no selection, insert [link](url) at the cursor
        if (selectionStart === selectionEnd) {
            textArea.value = textArea.value.substring(0, selectionStart) + '[display text](url)' + textArea.value.substring(selectionEnd);
            onChange(textArea.value);
            return;
        }
        // Otherwise, call padSelection
        padSelection('[', '](url)');
    }, [id, onChange, padSelection]);

    const insertBulletList = useCallback(() => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${id}`);
        const [lines, linesStart, linesEnd] = (getLinesOnSelection(textArea.value, selectionStart, selectionEnd) ?? [])
        const newValue = replaceText(textArea.value, lines.map(line => `* ${line}`).join('\n'), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [handleChange, id]);

    const insertNumberList = useCallback(() => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${id}`);
        const [lines, linesStart, linesEnd] = (getLinesOnSelection(textArea.value, selectionStart, selectionEnd) ?? [])
        const newValue = replaceText(textArea.value, lines.map((line, i) => `${i + 1}. ${line}`).join('\n'), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [handleChange, id]);

    const togglePreview = useCallback(() => { setIsPreviewOn(on => !on) }, []);

    // Mousedown prevents the textArea from removing its highlight when one 
    // of the buttons is clicked
    const handleMouseDown = useCallback((e) => { 
        if (isPreviewOn) return;
        // Get selection data
        const { selectionStart, selectionEnd } = getSelection(`markdown-input-${id}`);
        // Get target element id
        const targetId = e.target.id;
        // If the target is not the textArea, and the selection is not empty, then prevent default
        if (targetId !== `markdown-input-${id}` && selectionStart !== selectionEnd) { 
            console.log('preventing default');
            e.preventDefault() 
            e.stopPropagation();
        }
        console.log('mouse down', e);
        // e.preventDefault() 
    }, [id, isPreviewOn]);

    return (
        <Stack direction="column" spacing={0} onMouseDown={handleMouseDown}>
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
                        <IconButton
                            aria-describedby={`markdown-input-header-popover-${id}`}
                            disabled={disabled}
                            size="small"
                            onClick={openHeaderSelect}
                        >
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
                        <IconButton
                            disabled={disabled}
                            size="small"
                            onClick={bold}
                        >
                            <BoldIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Button for italic */}
                    <Tooltip title="Italic" placement="top">
                        <IconButton
                            disabled={disabled}
                            size="small"
                            onClick={italic}
                        >
                            <ItalicIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Button for strikethrough */}
                    <Tooltip title="Strikethrough" placement="top">
                        <IconButton
                            disabled={disabled}
                            size="small"
                            onClick={strikethrough}
                        >
                            <StrikethroughIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                    {/* Insert bulleted or numbered list selector */}
                    <Tooltip title="Insert list" placement="top">
                        <IconButton
                            aria-describedby={`markdown-input-list-popover-${id}`}
                            disabled={disabled}
                            size="small"
                            onClick={openListSelect}
                        >
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
                        <IconButton
                            disabled={disabled}
                            size="small"
                            onClick={insertLink}
                        >
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
                                    <InvisibleIcon fill={palette.primary.contrastText} /> :
                                    <VisibleIcon fill={palette.primary.contrastText} />
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
                            padding: '12px',
                            ...noSelect,
                        }}>
                            <Markdown>{internalValue}</Markdown>
                        </Box>
                    ) :
                    (
                        <textarea
                            id={`markdown-input-${id}`}
                            disabled={disabled}
                            placeholder={placeholder}
                            rows={minRows}
                            value={internalValue}
                            onChange={(e) => { handleChange(e.target.value) }}
                            style={{
                                padding: '16.5px 14px',
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
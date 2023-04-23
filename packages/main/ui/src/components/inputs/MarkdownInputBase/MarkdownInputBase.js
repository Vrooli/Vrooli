import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BoldIcon, Header1Icon, Header2Icon, Header3Icon, HeaderIcon, InvisibleIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListIcon, ListNumberIcon, RedoIcon, StrikethroughIcon, UndoIcon, VisibleIcon } from "@local/icons";
import { Box, IconButton, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import Markdown from "markdown-to-jsx";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { linkColors, noSelect } from "../../../styles";
import { useDebounce } from "../../../utils/hooks/useDebounce";
import { PubSub } from "../../../utils/pubsub";
var Headers;
(function (Headers) {
    Headers["H1"] = "h1";
    Headers["H2"] = "h2";
    Headers["H3"] = "h3";
    Headers["H4"] = "h4";
    Headers["H5"] = "h5";
    Headers["H6"] = "h6";
})(Headers || (Headers = {}));
const headerMarkdowns = {
    [Headers.H1]: "# ",
    [Headers.H2]: "## ",
    [Headers.H3]: "### ",
    [Headers.H4]: "#### ",
    [Headers.H5]: "##### ",
    [Headers.H6]: "###### ",
};
const dropDownButtonProps = ({
    borderRadius: 0,
    width: "48px",
    height: "48px",
});
const getLineStart = (text, selectionStart) => {
    if (selectionStart < 0 || selectionStart > text.length)
        return 0;
    return text.substring(0, selectionStart).lastIndexOf("\n") + 1;
};
const getLineEnd = (text, selectionStart) => {
    if (selectionStart < 0 || selectionStart > text.length)
        return text.length;
    return text.substring(selectionStart).indexOf("\n") + selectionStart;
};
const getLineAtIndex = (text, index) => {
    const start = getLineStart(text, index);
    const end = getLineEnd(text, index);
    const line = text.substring(start, end);
    return [line, start, end];
};
const getLinesAtRange = (text, start, end) => {
    const lineStart = getLineStart(text, start);
    const lineEnd = getLineEnd(text, end);
    const lines = text.substring(lineStart, lineEnd).split("\n");
    return [lines, lineStart, lineEnd];
};
const replaceText = (text, newText, start, end) => {
    return text.substring(0, start) + newText + text.substring(end);
};
const getSelection = (id) => {
    const textArea = document.getElementById(id);
    if (!textArea || !(textArea instanceof HTMLTextAreaElement))
        throw new Error(`Element not found: ${id}`);
    return { selectionStart: textArea.selectionStart, selectionEnd: textArea.selectionEnd, textArea };
};
export const MarkdownInputBase = ({ autoFocus = false, disabled = false, error = false, helperText, minRows = 4, name, onBlur, onChange, placeholder = "", tabIndex, value, sxs, }) => {
    const { palette } = useTheme();
    const changeStack = useRef([value]);
    const [changeStackIndex, setChangeStackIndex] = useState(0);
    const [internalValue, setInternalValue] = useState(value);
    useEffect(() => {
        const recentItems = changeStack.current.slice(Math.max(changeStack.current.length - 5, 0));
        if (value === "" || !recentItems.includes(value)) {
            setInternalValue(value);
        }
    }, [value]);
    const onChangeDebounced = useDebounce(onChange, 200);
    const undo = useCallback(() => {
        if (changeStackIndex > 0) {
            setChangeStackIndex(changeStackIndex - 1);
            setInternalValue(changeStack.current[changeStackIndex - 1]);
            onChangeDebounced(changeStack.current[changeStackIndex - 1]);
        }
    }, [changeStackIndex, onChangeDebounced]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.current.length > 0, [changeStackIndex]);
    const redo = useCallback(() => {
        if (changeStackIndex < changeStack.current.length - 1) {
            setChangeStackIndex(changeStackIndex + 1);
            setInternalValue(changeStack.current[changeStackIndex + 1]);
            onChangeDebounced(changeStack.current[changeStackIndex + 1]);
        }
    }, [changeStackIndex, onChangeDebounced]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.current.length - 1 && changeStack.current.length > 0, [changeStackIndex]);
    const handleChange = useCallback((updatedText) => {
        const newChangeStack = [...changeStack.current];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(updatedText);
        changeStack.current = newChangeStack;
        setChangeStackIndex(newChangeStack.length - 1);
        setInternalValue(updatedText);
        onChangeDebounced(updatedText);
    }, [changeStackIndex, onChangeDebounced]);
    const [isPreviewOn, setIsPreviewOn] = useState(false);
    const [headerAnchorEl, setHeaderAnchorEl] = useState(null);
    const openHeaderSelect = (event) => { setHeaderAnchorEl(event.currentTarget); };
    const closeHeader = () => { setHeaderAnchorEl(null); };
    const headerSelectOpen = Boolean(headerAnchorEl);
    const [listAnchorEl, setListAnchorEl] = useState(null);
    const openListSelect = (event) => { setListAnchorEl(event.currentTarget); };
    const closeList = () => { setListAnchorEl(null); };
    const listSelectOpen = Boolean(listAnchorEl);
    const insertHeader = useCallback((header) => {
        const { selectionStart, textArea } = getSelection(`markdown-input-${name}`);
        const startLine = getLineStart(textArea.value, selectionStart);
        const headerText = headerMarkdowns[header];
        if (textArea.value.substring(startLine, startLine + headerText.length) === headerText) {
            textArea.value = replaceText(textArea.value, "", startLine, startLine + headerText.length);
        }
        else {
            textArea.value = replaceText(textArea.value, headerText, startLine, startLine);
        }
        handleChange(textArea.value);
        closeHeader();
    }, [handleChange, name]);
    const padSelection = useCallback((padStart, padEnd) => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        if (selectionStart === selectionEnd) {
            PubSub.get().publishSnack({ messageKey: "NoTextSelected", severity: "Error" });
            return;
        }
        textArea.value = textArea.value.substring(0, selectionStart) + padStart + textArea.value.substring(selectionStart, selectionEnd) + padEnd + textArea.value.substring(selectionEnd);
        handleChange(textArea.value);
    }, [handleChange, name]);
    const strikethrough = useCallback(() => { padSelection("~~", "~~"); }, [padSelection]);
    const bold = useCallback(() => { padSelection("**", "**"); }, [padSelection]);
    const italic = useCallback(() => { padSelection("*", "*"); }, [padSelection]);
    const insertLink = useCallback(() => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        if (selectionStart === selectionEnd) {
            textArea.value = textArea.value.substring(0, selectionStart) + "[display text](url)" + textArea.value.substring(selectionEnd);
            onChange(textArea.value);
            return;
        }
        padSelection("[", "](url)");
    }, [name, onChange, padSelection]);
    const insertBulletList = useCallback(() => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        const [lines, linesStart, linesEnd] = (getLinesAtRange(textArea.value, selectionStart, selectionEnd) ?? []);
        const newValue = replaceText(textArea.value, lines.map(line => `* ${line}`).join("\n"), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [handleChange, name]);
    const insertNumberList = useCallback(() => {
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        const [lines, linesStart, linesEnd] = (getLinesAtRange(textArea.value, selectionStart, selectionEnd) ?? []);
        const newValue = replaceText(textArea.value, lines.map((line, i) => `${i + 1}. ${line}`).join("\n"), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [handleChange, name]);
    const togglePreview = useCallback(() => { setIsPreviewOn(on => !on); }, []);
    const handleMouseDown = useCallback((e) => {
        if (isPreviewOn)
            return;
        const { selectionStart, selectionEnd } = getSelection(`markdown-input-${name}`);
        const targetId = e.target.id;
        if (targetId !== `markdown-input-${name}` && selectionStart !== selectionEnd) {
            e.preventDefault();
            e.stopPropagation();
        }
    }, [isPreviewOn, name]);
    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.key === "Enter") {
                const { selectionStart, selectionEnd, value } = e.target;
                let [trimmedLine] = getLineAtIndex(value, selectionStart);
                trimmedLine = trimmedLine.trimStart();
                const isBullet = trimmedLine.startsWith("* ") || trimmedLine.trim()?.startsWith("- ");
                const isNumber = /^\d+\.\s/.test(trimmedLine);
                if (isBullet) {
                    e.preventDefault();
                    const { textArea } = getSelection(`markdown-input-${name}`);
                    textArea.value = replaceText(value, "\n* ", selectionStart, selectionEnd);
                    handleChange(textArea.value);
                }
                else if (isNumber) {
                    e.preventDefault();
                    const { textArea } = getSelection(`markdown-input-${name}`);
                    const currentLineNumber = Number(trimmedLine.match(/^\d+/)[0]);
                    textArea.value = replaceText(value, `\n${currentLineNumber + 1}. `, selectionStart, selectionEnd);
                    handleChange(textArea.value);
                }
            }
            else if (e.altKey && e.key === "1") {
                e.preventDefault();
                insertHeader(Headers.H1);
            }
            else if (e.altKey && e.key === "2") {
                e.preventDefault();
                insertHeader(Headers.H2);
            }
            else if (e.altKey && e.key === "3") {
                e.preventDefault();
                insertHeader(Headers.H3);
            }
            else if (e.ctrlKey && e.key === "b") {
                e.preventDefault();
                bold();
            }
            else if (e.ctrlKey && e.key === "i") {
                e.preventDefault();
                italic();
            }
            else if (e.ctrlKey && e.key === "S") {
                e.preventDefault();
                strikethrough();
            }
            else if (e.altKey && e.key === "4") {
                e.preventDefault();
                insertBulletList();
            }
            else if (e.altKey && e.key === "5") {
                e.preventDefault();
                insertNumberList();
            }
            else if (e.ctrlKey && e.key === "k") {
                e.preventDefault();
                insertLink();
            }
            else if (e.ctrlKey && (e.key === "y" || e.key === "Z")) {
                e.preventDefault();
                redo();
            }
            else if (e.ctrlKey && e.key === "z") {
                e.preventDefault();
                undo();
            }
            else if (e.altKey && e.key === "6") {
                e.preventDefault();
                togglePreview();
            }
        };
        const textarea = document.getElementById(`markdown-input-${name}`);
        if (!textarea)
            return;
        textarea.addEventListener("keydown", handleKeyDown);
        return () => textarea.removeEventListener("keydown", handleKeyDown);
    }, [bold, handleChange, insertBulletList, insertHeader, insertLink, insertNumberList, italic, name, redo, strikethrough, togglePreview, undo]);
    return (_jsxs(Stack, { direction: "column", spacing: 0, onMouseDown: handleMouseDown, children: [_jsxs(Box, { sx: {
                    display: "flex",
                    width: "100%",
                    padding: "0.5rem",
                    borderBottom: "1px solid #e0e0e0",
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    borderRadius: "0.5rem 0.5rem 0 0",
                    ...(sxs?.bar ?? {}),
                }, children: [_jsxs(Stack, { direction: "row", spacing: { xs: 0, sm: 0.5, md: 1 }, sx: { marginRight: "auto" }, children: [_jsx(Tooltip, { title: "Insert header (Title)", placement: "top", children: _jsx(IconButton, { "aria-describedby": `markdown-input-header-popover-${name}`, disabled: disabled, size: "small", onClick: openHeaderSelect, children: _jsx(HeaderIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Popover, { id: `markdown-input-header-popover-${name}`, open: headerSelectOpen, anchorEl: headerAnchorEl, onClose: closeHeader, anchorOrigin: {
                                    vertical: "bottom",
                                    horizontal: "center",
                                }, children: _jsxs(Stack, { direction: "row", spacing: 0, sx: {
                                        background: palette.primary.light,
                                        color: palette.primary.contrastText,
                                    }, children: [_jsx(Tooltip, { title: "Header 1 (ALT + 1)", placement: "top", children: _jsx(IconButton, { onClick: () => insertHeader(Headers.H1), sx: dropDownButtonProps, children: _jsx(Header1Icon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Header 2 (ALT + 2)", placement: "top", children: _jsx(IconButton, { onClick: () => insertHeader(Headers.H2), sx: dropDownButtonProps, children: _jsx(Header2Icon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Header 3 (ALT + 3)", placement: "top", children: _jsx(IconButton, { onClick: () => insertHeader(Headers.H3), sx: dropDownButtonProps, children: _jsx(Header3Icon, { fill: palette.primary.contrastText }) }) })] }) }), _jsx(Tooltip, { title: "Bold (CTRL + B)", placement: "top", children: _jsx(IconButton, { disabled: disabled, size: "small", onClick: bold, children: _jsx(BoldIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Italic (CTRL + I)", placement: "top", children: _jsx(IconButton, { disabled: disabled, size: "small", onClick: italic, children: _jsx(ItalicIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Strikethrough (CTRL + SHIFT + S)", placement: "top", children: _jsx(IconButton, { disabled: disabled, size: "small", onClick: strikethrough, children: _jsx(StrikethroughIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Insert list", placement: "top", children: _jsx(IconButton, { "aria-describedby": `markdown-input-list-popover-${name}`, disabled: disabled, size: "small", onClick: openListSelect, children: _jsx(ListIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Popover, { id: `markdown-input-list-popover-${name}`, open: listSelectOpen, anchorEl: listAnchorEl, onClose: closeList, anchorOrigin: {
                                    vertical: "bottom",
                                    horizontal: "center",
                                }, children: _jsxs(Stack, { direction: "row", spacing: 0, sx: {
                                        background: palette.primary.light,
                                        color: palette.primary.contrastText,
                                    }, children: [_jsx(Tooltip, { title: "Bulleted list (ALT + 4)", placement: "top", children: _jsx(IconButton, { onClick: insertBulletList, sx: dropDownButtonProps, children: _jsx(ListBulletIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: "Numbered list (ALT + 5)", placement: "top", children: _jsx(IconButton, { onClick: insertNumberList, sx: dropDownButtonProps, children: _jsx(ListNumberIcon, { fill: palette.primary.contrastText }) }) })] }) }), _jsx(Tooltip, { title: "Insert link (CTRL + K)", placement: "top", children: _jsx(IconButton, { disabled: disabled, size: "small", onClick: insertLink, children: _jsx(LinkIcon, { fill: palette.primary.contrastText }) }) })] }), _jsxs(Stack, { direction: "row", spacing: { xs: 0, sm: 0.5, md: 1 }, children: [(canUndo || canRedo) && _jsx(Tooltip, { title: canUndo ? "Undo (CTRL + Z)" : "", children: _jsx(IconButton, { id: "undo-button", disabled: !canUndo, onClick: undo, "aria-label": "Undo", size: "small", children: _jsx(UndoIcon, { fill: palette.primary.contrastText }) }) }), (canUndo || canRedo) && _jsx(Tooltip, { title: canRedo ? "Redo (CTRL + Y)" : "", children: _jsx(IconButton, { id: "redo-button", disabled: !canRedo, onClick: redo, "aria-label": "Redo", size: "small", children: _jsx(RedoIcon, { fill: palette.primary.contrastText }) }) }), _jsx(Tooltip, { title: isPreviewOn ? "Press to edit (ALT + 6)" : "Press to preview (ALT + 6)", placement: "top", children: _jsx(IconButton, { size: "small", onClick: togglePreview, children: isPreviewOn ?
                                        _jsx(InvisibleIcon, { fill: palette.primary.contrastText }) :
                                        _jsx(VisibleIcon, { fill: palette.primary.contrastText }) }) })] })] }), isPreviewOn ?
                (_jsx(Box, { sx: {
                        border: `1px solid ${error ? "red" : "black"}`,
                        borderRadius: "0 0 0.5rem 0.5rem",
                        borderTop: "none",
                        padding: "12px",
                        ...noSelect,
                        ...linkColors(palette),
                    }, children: _jsx(Markdown, { children: internalValue }) })) :
                (_jsx("textarea", { id: `markdown-input-${name}`, autoFocus: autoFocus, disabled: disabled, name: name, placeholder: placeholder, rows: minRows, value: internalValue, onBlur: onBlur, onChange: (e) => { handleChange(e.target.value); }, tabIndex: tabIndex, style: {
                        padding: "16.5px 14px",
                        minWidth: "-webkit-fill-available",
                        maxWidth: "-webkit-fill-available",
                        minHeight: "50px",
                        maxHeight: "800px",
                        background: "transparent",
                        borderColor: error ? "red" : "unset",
                        borderRadius: "0 0 0.5rem 0.5rem",
                        borderTop: "none",
                        fontFamily: "inherit",
                        fontSize: "inherit",
                        color: palette.text.primary,
                        ...(sxs?.textArea ?? {}),
                    } })), helperText &&
                _jsx(Typography, { variant: "body1", sx: { color: "red", paddingTop: 1 }, children: helperText })] }));
};
//# sourceMappingURL=MarkdownInputBase.js.map
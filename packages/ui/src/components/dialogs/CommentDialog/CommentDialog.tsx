import { AppBar, Box, Button, Dialog, Stack, Typography, useTheme } from "@mui/material";
import { MarkdownInput } from "components/inputs";
import { useCallback, useMemo, useState } from "react";
import { getTranslation } from "utils";
import { PopoverWithArrow } from "../PopoverWithArrow/PopoverWithArrow";
import { UpTransition } from "../transitions";
import { CommentDialogProps } from "../types"


/**
 * Dialog for creating/updating a comment. 
 * Only used on mobile; desktop displays MarkdownInput at top of 
 * CommentContainer
 */
export const CommentDialog = ({
    errorText,
    handleSubmit,
    handleClose,
    isAdding,
    isOpen,
    language,
    onTranslationChange,
    parent,
    text,
    zIndex,
}: CommentDialogProps) => {
    const { palette } = useTheme();
    console.log('comment dialog', errorText);

    const { parentText } = useMemo(() => {
        const { text } = getTranslation(parent, [language]);
        return {
            parentText: text,
        };
    }, [language, parent]);

    // Errors popup
    const [errorAnchorEl, setErrorAnchorEl] = useState<any | null>(null);
    const openError = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target)
    }, []);
    const closeError = useCallback(() => {
        setErrorAnchorEl(null);
    }, []);

    const onSubmit = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        // If formik invalid, display errors in popup
        if (errorText.length > 0) openError(ev);
        else handleSubmit();
    }, [errorText.length, openError, handleSubmit]);

    if (!isOpen) return null;
    return (
        <Dialog
            fullScreen
            id="create-comment-dialog"
            onClose={handleClose}
            open={isOpen}
            TransitionComponent={UpTransition}
            sx={{
                zIndex: zIndex + 1,
                '& .MuiDialog-paper': {
                    background: palette.background.default,
                    color: palette.background.textPrimary,
                },
            }}
        >
            {/* Errors popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeError}
                sxs={{
                    root: {
                        // Remove horizontal spacing for list items
                        '& ul': {
                            paddingInlineStart: '20px',
                            margin: '8px',
                        }
                    }
                }}
            >
                {errorText}
            </PopoverWithArrow>
            {/* App bar with Cancel, title, and Submit */}
            <AppBar sx={{ position: 'relative', background: palette.primary.dark }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 1 }}>
                    {/* Cancel button */}
                    <Button variant="text" onClick={handleClose}>
                        Cancel
                    </Button>
                    {/* Title */}
                    <Typography variant="h6" component="div" sx={{ flexGrow: 1, textAlign: 'center' }}>
                        {isAdding ? 'Add comment' : 'Edit comment'}
                    </Typography>
                    {/* Submit button */}
                    <Box onClick={onSubmit}>
                        <Button
                            disabled={errorText.length > 0}
                            variant="text"
                        >
                            Submit
                        </Button>
                    </Box>
                </Box>
            </AppBar>
            {/* Main content */}
            <Stack direction="column" spacing={0}>
                {/* Input for comment */}
                <MarkdownInput
                    id="add-comment"
                    placeholder="Please be nice to each other."
                    value={text}
                    minRows={6}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                    error={text.length > 0 && Boolean(errorText)}
                    helperText={text.length > 0 ? errorText : ''}
                    sxs={{
                        bar: {
                            borderRadius: 0,
                            background: palette.primary.main,
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: 'none',
                        }
                    }}
                />
                {/* Display parent underneath */}
                {parent && (
                    <Box sx={{
                        backgroundColor: palette.background.paper,
                    }}>
                        <Typography variant="body2">{parentText}</Typography>
                    </Box>
                )}
            </Stack>
        </Dialog >
    )
}
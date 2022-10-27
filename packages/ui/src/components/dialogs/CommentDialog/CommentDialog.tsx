import { AppBar, Box, Button, Dialog, Stack, Typography, useTheme } from "@mui/material";
import { MarkdownInput } from "components/inputs";
import { useMemo } from "react";
import { getTranslation } from "utils";
import { UpTransition } from "../transitions";
import { CommentDialogProps } from "../types"


/**
 * Dialog for creating/updating a comment. 
 * Only used on mobile; desktop displays MarkdownInput at top of 
 * CommentContainer
 */
export const CommentDialog = ({
    errors,
    errorText,
    handleClose,
    isAdding,
    isOpen,
    language,
    onTranslationBlur,
    onTranslationChange,
    parent,
    text,
    touchedText,
    zIndex,
}: CommentDialogProps) => {
    const { palette } = useTheme();
    console.log('comment dialog', touchedText, errorText);

    const { parentText } = useMemo(() => {
        const { text } = getTranslation(parent, [language]);
        return {
            parentText: text,
        };
    }, [language, parent]);

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => value !== null && value !== undefined), [errors]);

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
                    <Button disabled={hasErrors} variant="text" onClick={handleClose}>
                        Submit
                    </Button>
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
                    error={touchedText && Boolean(errorText)}
                    helperText={touchedText ? errorText : null}
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
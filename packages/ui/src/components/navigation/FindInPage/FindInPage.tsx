import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Dialog,
    DialogContent,
    IconButton,
    Palette,
    TextField,
    Tooltip,
    useTheme,
} from '@mui/material';
import { PubSub } from 'utils';
import { Stack } from '@mui/system';
import { ArrowDownIcon, ArrowUpIcon, CaseSensitiveIcon, CloseIcon, RegexIcon, WholeWordIcon } from 'assets/img';

const commonButtonSx = (palette: Palette, isActivated?: boolean) => ({
    background: isActivated ? palette.secondary.dark : 'transparent',
    borderRadius: '0',
    color: 'inherit',
    width: '40px',
    height: '40px',
    '&:hover': {
        background: isActivated ? palette.secondary.dark : 'transparent',
        filter: 'brightness(120%)',
    },
})

const commonIconProps = (palette: Palette) => ({
    fill: palette.background.textPrimary,
})

const highlightText = (
    searchString: string,
    isCaseSensitive: boolean,
    isWholeWord: boolean,
    isRegex: boolean,
) => {
    // Read page data
    const innerText = document.body.innerText;
    let innerHtml = document.body.innerHTML;
    // Remove special characters from search string
    let convertedSearchString = searchString.replace(/[#-.]|[[-^]|[?|{}]/g, '\\$&');
    // If whole word, wrap \b around search string
    if (isWholeWord) convertedSearchString = `\\b${convertedSearchString}\\b`;
    // Create regex from search string
    const regex = new RegExp(convertedSearchString, isCaseSensitive ? '' : 'i');
    // Find all matches
    const matches = innerText.matchAll(regex);
    for (const match of matches) {
        innerHtml = innerHtml.replace(match[0], `<span style="background-color: yellow">${match[0]}</span>`);
    }
    // Update the body's innerHTML
    document.body.innerHTML = innerHtml;
}

const FindInPage = () => {
    const { palette } = useTheme();

    const [open, setOpen] = useState(false);
    const close = useCallback(() => setOpen(false), []);

    const [isCaseSensitive, setIsCaseSensitive] = useState(false);
    const [isWholeWord, setIsWholeWord] = useState(false);
    const [isRegex, setIsRegex] = useState(false);

    const [results, setResults] = useState<string[]>([]);
    const [resultIndex, setResultIndex] = useState(0);
    const [searchString, setSearchString] = useState<string>('');

    const onCaseSensitiveChange = useCallback(() => setIsCaseSensitive(o => !o), []);
    const onWholeWordChange = useCallback(() => setIsWholeWord(o => !o), []);
    const onRegexChange = useCallback(() => setIsRegex(o => !o), []);

    const onPrevious = useCallback(() => setResultIndex(o => {
        if (o > 0) return o - 1;
        else return results.length - 1;
    }), [results.length]);
    const onNext = useCallback(() => setResultIndex(o => {
        if (o < results.length - 1) return o + 1;
        else return 0;
    }), [results.length]);

    const onSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        // Update search string
        setSearchString(e.target.value);
        // Calculate results
        highlightText(e.target.value, isCaseSensitive, isWholeWord, isRegex);
    }, [isCaseSensitive, isRegex, isWholeWord]);

    useEffect(() => {
        let dialogSub = PubSub.get().subscribeFindInPage(() => {
            setOpen(o => {
                // If turning off, reset search values (but keep case sensitive and other buttons the same)
                if (o) {
                    setSearchString('');
                    setResults([]);
                    setResultIndex(0);
                }
                return !o;
            });
        });
        return () => { PubSub.get().unsubscribe(dialogSub) };
    }, [])

    return (
        <Dialog
            open={open}
            disableScrollLock={true}
            sx={{
                '& .MuiDialog-container': {
                    color: 'transparent'
                },
                '& .MuiDialog-paper': {
                    border: palette.mode === 'dark' ? `1px solid white` : 'unset',
                    minWidth: 'min(100%, 400px)',
                    position: 'absolute',
                    top: '0%',
                    right: '0%',
                    overflowY: 'visible',
                }
            }}
        >
            <DialogContent sx={{
                background: palette.background.default,
                position: 'relative',
                overflowY: 'visible',
            }}>
                <Stack direction="row">
                    <Stack direction="row" sx={{
                        background: palette.background.paper,
                        borderRadius: '4px',
                        border: `1px solid ${palette.background.textPrimary}`,
                    }}>
                        {/* Search bar */}
                        <TextField
                            id="command-palette-search"
                            autoFocus={true}
                            placeholder='Find in page...'
                            value={searchString}
                            onChange={onSearchChange}
                            size="small"
                            sx={{
                                width: '100%',
                                border: 'none',
                                borderRight: `1px solid ${palette.background.textPrimary}`,
                            }}
                        />
                        {/* Buttons for case-sensitive, match whole word, and regex */}
                        <Box display="flex" alignItems="center">
                            <Tooltip title="Match case (Alt+C)">
                                <IconButton aria-label="case-sensitive" sx={commonButtonSx(palette, isCaseSensitive)} onClick={onCaseSensitiveChange}>
                                    <CaseSensitiveIcon {...commonIconProps(palette)} />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Match whole word (Alt+W)">
                                <IconButton aria-label="match whole word" sx={commonButtonSx(palette, isWholeWord)} onClick={onWholeWordChange}>
                                    <WholeWordIcon {...commonIconProps(palette)} />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Use regular expression (Alt+R)">
                                <IconButton aria-label="match regex" sx={commonButtonSx(palette, isRegex)} onClick={onRegexChange}>
                                    <RegexIcon {...commonIconProps(palette)} />
                                </IconButton>
                            </Tooltip>
                        </Box>
                    </Stack>
                    {/* Up and down arrows, and close icon */}
                    <Box display="flex" alignItems="center" justifyContent="flex-end">
                        <Tooltip title="Previous result (Shift+Enter)">
                            <IconButton aria-label="previous result" sx={commonButtonSx(palette)} onClick={onPrevious}>
                                <ArrowUpIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title="Next result (Enter)">
                            <IconButton aria-label="next result" sx={commonButtonSx(palette)} onClick={onNext}>
                                <ArrowDownIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title="Close">
                            <IconButton aria-label="close" sx={commonButtonSx(palette)} onClick={close}>
                                <CloseIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                    </Box>
                </Stack>
            </DialogContent>
        </Dialog >
    );
}

export { FindInPage };
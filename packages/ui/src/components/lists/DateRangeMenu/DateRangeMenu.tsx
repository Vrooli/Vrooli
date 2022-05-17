import { useState } from 'react';
import { Box, Button, Popover, Stack, TextField, Typography, useTheme } from "@mui/material";
import { DateRangeMenuProps } from "../types";
import { LocalizationProvider, MobileDatePicker } from '@mui/lab';
import AdapterDateFns from '@mui/lab/AdapterDateFns';

export const DateRangeMenu = ({
    anchorEl,
    onClose,
    onSubmit,
}: DateRangeMenuProps) => {
    const { palette } = useTheme();

    const open = Boolean(anchorEl);

    const [after, setAfter] = useState<Date | null>(null)
    const [before, setBefore] = useState<Date | null>(null)

    const handleAfterChange = (date: Date | null) => setAfter(date);
    const handleBeforeChange = (date: Date | null) => setBefore(date)

    return (
        <Popover
            id='date-range-popover'
            anchorEl={anchorEl}
            open={open}
            onClose={onClose}
            disableScrollLock={true}
        >
            <Typography textAlign="center" p={1} sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
            }}>Select date range</Typography>
            <LocalizationProvider dateAdapter={AdapterDateFns}>
                <Box p={2}>
                    <Stack direction="column">
                        <MobileDatePicker
                            label="Start date"
                            inputFormat="dd/MM/yyyy"
                            value={after}
                            onChange={handleAfterChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <MobileDatePicker
                            label="End date"
                            inputFormat="dd/MM/yyyy"
                            value={before}
                            onChange={handleBeforeChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <Button
                            type="submit"
                            fullWidth
                            onClick={() => { onSubmit(after, before); onClose() }}
                        >Go</Button>
                    </Stack>
                </Box>
            </LocalizationProvider>
        </Popover>
    )
}
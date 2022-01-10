import { useState } from 'react';
import { Button, Popover, Stack, TextField, Typography } from "@mui/material";
import { DateRangeMenuProps } from "../types";
import { LocalizationProvider, MobileDatePicker } from '@mui/lab';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { centeredText } from 'styles';

export const DateRangeMenu = ({
    anchorEl,
    onClose,
    onSubmit,
}: DateRangeMenuProps) => {
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
            <Typography sx={{ ...centeredText }}>Select date range</Typography>
            <LocalizationProvider dateAdapter={AdapterDateFns}>
                <Stack direction="column">
                    <MobileDatePicker
                        label="Start date"
                        inputFormat="dd/MM/yyyy"
                        value={after}
                        onChange={handleAfterChange}
                        renderInput={(params) => <TextField {...params} />}
                    />
                    <MobileDatePicker
                        label="End date"
                        inputFormat="dd/MM/yyyy"
                        value={before}
                        onChange={handleBeforeChange}
                        renderInput={(params) => <TextField {...params} />}
                    />
                    <Button
                        type="submit"
                        fullWidth
                        onClick={() => { onSubmit(after, before); onClose() }}
                    >Go</Button>
                </Stack>
            </LocalizationProvider>
        </Popover>
    )
}
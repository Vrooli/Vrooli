import { LocalizationProvider, MobileDatePicker } from '@mui/lab';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { Box, Button, Popover, Stack, TextField, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { DateRangeMenuProps } from "../types";

export const DateRangeMenu = ({
    anchorEl,
    minDate,
    maxDate,
    onClose,
    onSubmit,
    range,
    strictIntervalRange
}: DateRangeMenuProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const open = Boolean(anchorEl);

    // Internal state for range.after and range.before
    const [after, setAfter] = useState<Date | undefined>(range?.after ?? minDate);
    const [before, setBefore] = useState<Date | undefined>(range?.before ?? maxDate)
    const handleAfterChange = useCallback((date: Date | null) => { setAfter(date ?? minDate) }, [minDate]);
    const handleBeforeChange = useCallback((date: Date | null) => { setBefore(date ?? maxDate) }, [maxDate]);

    useEffect(() => {
        setAfter(range?.after ?? minDate);
        setBefore(range?.before ?? maxDate);
    }, [range, minDate, maxDate]);

    useEffect(() => {
        let changedAfter = after ?? minDate ?? new Date();
        if (changedAfter < (minDate ?? 0)) {
            changedAfter = minDate ?? new Date(0);
        }
        const latestBefore = new Date(Date.now() - (strictIntervalRange ?? 0));
        if (changedAfter > latestBefore) {
            changedAfter = latestBefore;
        }
        // Only update after if it is different from the new calculated value. 
        // Ignore changes smaller than 1 second
        if (changedAfter.getTime() - (after?.getTime() ?? 0) > 1000) {
            setAfter(changedAfter);
            // Only update before if it is different from the new calculated value
            if (strictIntervalRange && before !== new Date(changedAfter.getTime() + strictIntervalRange)) {
                setBefore(new Date(changedAfter.getTime() + strictIntervalRange));
            }
        }
    }, [after, before, minDate, strictIntervalRange]);

    useEffect(() => {
        let changedBefore = before ?? maxDate ?? new Date();
        if (after && changedBefore < after) {
            changedBefore = after;
        }
        if (changedBefore > new Date()) {
            changedBefore = new Date();
        }
        // Only update before if it is different from the new calculated value.
        // Ignore changes smaller than 1 second
        if (changedBefore.getTime() !== (before?.getTime() ?? 0)) {
            setBefore(changedBefore);
        }
    }, [after, before, maxDate]);

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
            }}>{t(`SelectDateRange`)}</Typography>
            <LocalizationProvider dateAdapter={AdapterDateFns}>
                <Box p={2}>
                    <Stack direction="column">
                        <MobileDatePicker
                            label={t(`Start`)}
                            inputFormat="dd/MM/yyyy"
                            value={after}
                            onChange={handleAfterChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <MobileDatePicker
                            label={t(`End`)}
                            inputFormat="dd/MM/yyyy"
                            value={before}
                            onChange={handleBeforeChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <Button
                            type="submit"
                            fullWidth
                            onClick={() => { onSubmit(after, before); onClose() }}
                        >{t(`Ok`)}</Button>
                    </Stack>
                </Box>
            </LocalizationProvider>
        </Popover>
    )
}
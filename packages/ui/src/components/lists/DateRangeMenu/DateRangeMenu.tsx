import { useCallback, useEffect, useMemo, useState } from 'react';
import { Box, Button, Popover, Stack, TextField, Typography, useTheme } from "@mui/material";
import { DateRangeMenuProps } from "../types";
import { LocalizationProvider, MobileDatePicker } from '@mui/lab';
import AdapterDateFns from '@mui/lab/AdapterDateFns';
import { useTranslation } from 'react-i18next';
import { getUserLanguages } from 'utils';

export const DateRangeMenu = ({
    anchorEl,
    minDate,
    maxDate,
    onClose,
    onSubmit,
    range,
    session,
    strictIntervalRange
}: DateRangeMenuProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

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
        console.log('in drm 1', after, minDate, new Date());
        let changedAfter = after ?? minDate ?? new Date();
        if (changedAfter < (minDate ?? 0)) {
            console.log('in drm 2');
            changedAfter = minDate ?? new Date(0);
        }
        const latestBefore = new Date(Date.now() - (strictIntervalRange ?? 0));
        console.log('in drm 3', changedAfter, latestBefore);
        if (changedAfter > latestBefore) {
            console.log('in drm 4');
            changedAfter = latestBefore;
        }
        // Only update after if it is different from the new calculated value. 
        // Ignore changes smaller than 1 second
        if (changedAfter.getTime() - (after?.getTime() ?? 0) > 1000) {
            console.log('in drm 5', changedAfter.getTime() - (after?.getTime() ?? 0));
            setAfter(changedAfter);
            // Only update before if it is different from the new calculated value
            if (strictIntervalRange && before !== new Date(changedAfter.getTime() + strictIntervalRange)) {
                console.log('in drm 6', changedAfter.getTime() - (after?.getTime() ?? 0));
                setBefore(new Date(changedAfter.getTime() + strictIntervalRange));
            }
        }
    }, [after, before, minDate, strictIntervalRange]);

    useEffect(() => {
        console.log('in drm 7', before, maxDate, new Date());
        let changedBefore = before ?? maxDate ?? new Date();
        if (after && changedBefore < after) {
            console.log('in drm 8');
            changedBefore = after;
        }
        if (changedBefore > new Date()) {
            console.log('in drm 9');
            changedBefore = new Date();
        }
        // Only update before if it is different from the new calculated value.
        // Ignore changes smaller than 1 second
        if (changedBefore.getTime() !== (before?.getTime() ?? 0)) {
            console.log('in drm 10', changedBefore.getTime() - (before?.getTime() ?? 0))
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
            }}>{t(`common:SelectDateRange`, { lng })}</Typography>
            <LocalizationProvider dateAdapter={AdapterDateFns}>
                <Box p={2}>
                    <Stack direction="column">
                        <MobileDatePicker
                            label={t(`common:Start`, { lng })}
                            inputFormat="dd/MM/yyyy"
                            value={after}
                            onChange={handleAfterChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <MobileDatePicker
                            label={t(`common:End`, { lng })}
                            inputFormat="dd/MM/yyyy"
                            value={before}
                            onChange={handleBeforeChange}
                            renderInput={(params) => <TextField {...params} sx={{ marginBottom: 1 }} />}
                        />
                        <Button
                            type="submit"
                            fullWidth
                            onClick={() => { onSubmit(after, before); onClose() }}
                        >{t(`common:Ok`, { lng })}</Button>
                    </Stack>
                </Box>
            </LocalizationProvider>
        </Popover>
    )
}
import { Button, Grid, Popover, Stack, TextField } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { fromDatetimeLocal, toDatetimeLocal } from "../../../utils/shape/general";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { DateRangeMenuProps } from "../types";

export const DateRangeMenu = ({
    anchorEl,
    minDate,
    maxDate,
    onClose,
    onSubmit,
    range,
    strictIntervalRange,
}: DateRangeMenuProps) => {
    const { t } = useTranslation();

    const open = Boolean(anchorEl);

    // Internal state for range.after and range.before
    const [after, setAfter] = useState<Date | undefined>(range?.after ?? minDate);
    const [before, setBefore] = useState<Date | undefined>(range?.before ?? maxDate);
    const handleAfterChange = useCallback((date: Date | null) => { setAfter(date ?? minDate); }, [minDate]);
    const handleBeforeChange = useCallback((date: Date | null) => { setBefore(date ?? maxDate); }, [maxDate]);

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
            <TopBar
                display="dialog"
                onClose={onClose}
                titleData={{
                    titleKey: "SelectDateRange",
                }}
            />
            <Stack direction="column" spacing={2} m={2}>
                <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                        <TextField
                            fullWidth
                            name="start"
                            label={t("Start")}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            value={after ? toDatetimeLocal(after) : ""}
                            onChange={(e) => handleAfterChange(fromDatetimeLocal(e.target.value))}
                        />
                    </Grid>
                    <Grid item xs={12} sm={6}>
                        <TextField
                            fullWidth
                            name="end"
                            label={t("End")}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            value={before ? toDatetimeLocal(before) : ""}
                            onChange={(e) => handleBeforeChange(fromDatetimeLocal(e.target.value))}
                        />
                    </Grid>
                </Grid>
                <Button
                    type="submit"
                    fullWidth
                    onClick={() => { onSubmit(after, before); onClose(); }}
                >{t("Ok")}</Button>
            </Stack>
        </Popover>
    );
};
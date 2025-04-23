import { fromDatetimeLocal, toDatetimeLocal } from "@local/shared";
import { Box, Button, Grid, Popover } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { TopBar } from "../../navigation/TopBar.js";
import { DateRangeMenuProps } from "../types.js";

export function DateRangeMenu({
    anchorEl,
    minDate = new Date(0),
    maxDate = new Date(),
    onClose,
    onSubmit,
    range,
    strictIntervalRange,
}: DateRangeMenuProps) {
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
                display="Dialog"
                onClose={onClose}
                title={t("SelectDateRange")}
                variant="subheader"
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                gap: 2,
                margin: 2,
            }}>
                <Grid container>
                    <Grid item xs={12} sm={6} p={1}>
                        <TextInput
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
                    <Grid item xs={12} sm={6} p={1}>
                        <TextInput
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
                    variant="contained"
                    sx={{
                        marginLeft: 1,
                        marginRight: 1,
                        width: "-webkit-fill-available",
                    }}
                >{t("Ok")}</Button>
            </Box>
        </Popover >
    );
}

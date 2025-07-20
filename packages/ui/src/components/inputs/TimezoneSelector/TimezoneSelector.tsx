import { default as ListItem, default as Popover, default as Stack, default as Typography, useTheme } from "@mui/material";
import { HOURS_1_M, MINUTES_1_MS, MINUTES_1_S } from "@vrooli/shared";
import { useField } from "formik";
import React, { forwardRef, useCallback, useMemo, useState } from "react";
import { FixedSizeList } from "react-window";
import { usePopover } from "../../../hooks/usePopover.js";
import { IconCommon } from "../../../icons/Icons.js";
import { IconButton } from "../../buttons/IconButton.js";
import { MenuTitle } from "../../dialogs/MenuTitle/MenuTitle.js";
import { TextInputBase } from "../TextInput/TextInput.js";
import { type TimezoneSelectorBaseProps, type TimezoneSelectorFormikProps } from "../types.js";

function formatOffset(offset: number) {
    const sign = offset > 0 ? "-" : "+";
    const hours = Math.abs(Math.floor(offset / HOURS_1_M));
    const minutes = Math.abs(offset % MINUTES_1_S);
    return `${sign}${hours.toString().padStart(2, "0")}:${minutes.toString().padStart(2, "0")}`;
}

function getTimezoneOffset(timezone) {
    const now = new Date();
    const localTime = now.getTime();
    const localOffset = now.getTimezoneOffset() * MINUTES_1_MS;
    const utcTime = localTime + localOffset;

    const targetTime = new Date(now.toLocaleString("en-US", { timeZone: timezone }));
    const targetOffset = targetTime.getTime() - utcTime;

    // Divide by the number of milliseconds in a minute, then round to the nearest tenth
    const roundedOffset = Math.round((targetOffset / MINUTES_1_MS) * 10) / 10;

    return -roundedOffset;
}

const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

/**
 * Base timezone selector component without Formik integration.
 * This is the pure visual component that handles all styling and interaction logic.
 */
export const TimezoneSelectorBase = forwardRef<HTMLInputElement, TimezoneSelectorBaseProps>(({
    value,
    onChange,
    onBlur,
    error,
    helperText,
    label,
    isRequired,
    disabled,
    name,
    ...props
}, ref) => {
    const { palette } = useTheme();

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    const timezones = useMemo(() => {
        const allTimezones = (Intl as any).supportedValuesOf("timeZone") as string[];
        if (searchString.length > 0) {
            return allTimezones.filter(tz => tz.toLowerCase().includes(searchString.toLowerCase()));
        }
        return allTimezones;
    }, [searchString]);

    const timezoneData = useMemo(() => {
        return timezones.map((timezone) => {
            const timezoneOffset = getTimezoneOffset(timezone);
            const formattedOffset = formatOffset(timezoneOffset);
            return { timezone, timezoneOffset, formattedOffset };
        });
    }, [timezones]);

    const [anchorEl, onOpen, onClose, isOpen] = usePopover();
    const handleClose = useCallback(() => {
        setSearchString("");
        onClose();
    }, [onClose]);

    const handleTimezoneSelect = useCallback((timezone: string) => {
        onChange(timezone);
        handleClose();
    }, [onChange, handleClose]);

    return (
        <>
            {/* Popover virtualized list */}
            <Popover
                open={isOpen}
                anchorEl={anchorEl}
                onClose={handleClose}
                anchorOrigin={anchorOrigin}
                transformOrigin={transformOrigin}
                sx={{
                    "& .MuiPopover-paper": {
                        width: "100%",
                        maxWidth: 500,
                    },
                }}
            >
                <MenuTitle
                    ariaLabel={""}
                    title={"Select Timezone"}
                    onClose={handleClose}
                />
                <Stack direction="column" spacing={2} p={2}>
                    <TextInputBase
                        placeholder="Enter timezone..."
                        autoFocus={true}
                        value={searchString}
                        onChange={updateSearchString}
                    />
                    {/* TODO Remove this once react-window is updated */}
                    <FixedSizeList
                        height={600}
                        width="100%"
                        itemSize={46}
                        itemCount={timezones.length}
                        overscanCount={5}
                    >
                        {({ index, style }) => {
                            const { timezone, formattedOffset } = timezoneData[index];

                            return (
                                <ListItem
                                    button
                                    key={index}
                                    style={style}
                                    onClick={() => handleTimezoneSelect(timezone)}
                                    disabled={disabled}
                                >
                                    <Stack direction="row" justifyContent="space-between" alignItems="center" width="100%">
                                        <Typography>{timezone}</Typography>
                                        <Typography>{` (UTC${formattedOffset})`}</Typography>
                                    </Stack>
                                </ListItem>
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Text input that looks like a selector */}
            <TextInputBase
                {...props}
                ref={ref}
                name={name}
                label={label}
                value={value}
                onClick={onOpen}
                onBlur={onBlur}
                error={error}
                helperText={helperText}
                isRequired={isRequired}
                disabled={disabled}
                InputProps={{
                    readOnly: true,
                    endAdornment: (
                        <IconButton
                            aria-label="timezone-select"
                            size="small"
                            disabled={disabled}
                        >
                            {isOpen ?
                                <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="ArrowDropUp"
                                />
                                : <IconCommon
                                    decorative
                                    fill={palette.background.textPrimary}
                                    name="ArrowDropDown"
                                />
                            }
                        </IconButton>
                    ),
                }}
                spellCheck={false}
            />
        </>
    );
});

TimezoneSelectorBase.displayName = "TimezoneSelectorBase";

/**
 * Formik-integrated timezone selector component.
 * Automatically connects to Formik context using the field name.
 * 
 * @example
 * ```tsx
 * // Inside a Formik form
 * <TimezoneSelector name="timezone" label="Timezone" />
 * 
 * // With validation
 * <TimezoneSelector 
 *   name="userTimezone" 
 *   label="Your Timezone"
 *   validate={(value) => !value ? "Timezone is required" : undefined}
 * />
 * ```
 */
export const TimezoneSelector = forwardRef<HTMLInputElement, TimezoneSelectorFormikProps>(({
    name,
    validate,
    ...props
}, ref) => {
    const [field, meta, helpers] = useField({ name, validate });

    const handleChange = useCallback((value: string) => {
        helpers.setValue(value);
    }, [helpers]);

    return (
        <TimezoneSelectorBase
            {...props}
            ref={ref}
            name={name}
            value={field.value || ""}
            onChange={handleChange}
            onBlur={field.onBlur}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
        />
    );
});

TimezoneSelector.displayName = "TimezoneSelector";

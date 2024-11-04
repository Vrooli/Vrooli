import { MINUTES_1_MS } from "@local/shared";
import { IconButton, ListItem, Popover, Stack, Typography, useTheme } from "@mui/material";
import { MenuTitle } from "components/dialogs/MenuTitle/MenuTitle";
import { useField } from "formik";
import { usePopover } from "hooks/usePopover";
import { ArrowDropDownIcon, ArrowDropUpIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { FixedSizeList } from "react-window";
import { TextInput } from "../TextInput/TextInput";
import { TimezoneSelectorProps } from "../types";

function formatOffset(offset: number) {
    const sign = offset > 0 ? "-" : "+";
    const hours = Math.abs(Math.floor(offset / 60));
    const minutes = Math.abs(offset % 60);
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

export function TimezoneSelector({
    onChange,
    ...props
}: TimezoneSelectorProps) {
    const { palette } = useTheme();

    const [field, , helpers] = useField(props.name);
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
                    <TextInput
                        placeholder="Enter timezone..."
                        autoFocus={true}
                        value={searchString}
                        onChange={updateSearchString}
                    />
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
                                    onClick={() => {
                                        helpers.setValue(timezone);
                                        onChange?.(timezone);
                                        handleClose();
                                    }}
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
            <TextInput
                fullWidth={false}
                sx={{
                    "& .MuiInputBase-root": {
                        width: "max-content",
                    },
                }}
                {...props}
                value={field.value}
                onClick={onOpen}
                InputProps={{
                    endAdornment: (
                        <IconButton size="small" aria-label="timezone-select">
                            {isOpen ?
                                <ArrowDropUpIcon fill={palette.background.textPrimary} /> :
                                <ArrowDropDownIcon fill={palette.background.textPrimary} />
                            }
                        </IconButton>
                    ),
                }}
            />
        </>
    );
}

import { IconButton, ListItem, Popover, Stack, TextField, Typography, useTheme } from '@mui/material';
import { ArrowDropDownIcon, ArrowDropUpIcon } from '@shared/icons';
import { MenuTitle } from 'components/dialogs/MenuTitle/MenuTitle';
import { useField } from 'formik';
import { useCallback, useMemo, useState } from 'react';
import { FixedSizeList } from 'react-window';
import { TimezoneSelectorProps } from '../types';

const formatOffset = (offset) => {
    const sign = offset > 0 ? '-' : '+';
    const hours = Math.abs(Math.floor(offset / 60));
    const minutes = Math.abs(offset % 60);
    return `${sign}${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
};

const getTimezoneOffset = (timezone) => {
    const now = new Date();
    const localTime = now.getTime();
    const localOffset = now.getTimezoneOffset() * 60 * 1000;
    const utcTime = localTime + localOffset;

    const targetTime = new Date(now.toLocaleString('en-US', { timeZone: timezone }));
    const targetOffset = targetTime.getTime() - utcTime;

    // Divide by the number of milliseconds in a minute, then round to the nearest tenth
    const roundedOffset = Math.round((targetOffset / (60 * 1000)) * 10) / 10;

    return -roundedOffset;
};

export const TimezoneSelector = ({
    onChange,
    ...props
}: TimezoneSelectorProps) => {
    const { palette } = useTheme();

    const [field, , helpers] = useField(props.name);
    const [searchString, setSearchString] = useState('');
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    const timezones = useMemo(() => {
        const allTimezones = (Intl as any).supportedValuesOf('timeZone') as string[];
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


    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    }, []);
    const onClose = useCallback(() => {
        setSearchString('');
        setAnchorEl(null);
    }, []);

    return (
        <>
            {/* Popover virtualized list */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                sx={{
                    '& .MuiPopover-paper': {
                        width: '100%',
                        maxWidth: 500,
                    },
                }}
            >
                <MenuTitle
                    ariaLabel={''}
                    title={'Select Timezone'}
                    onClose={onClose}
                />
                <Stack direction="column" spacing={2} p={2}>
                    <TextField
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
                            const { timezone, formattedOffset } = timezoneData[index]

                            return (
                                <ListItem
                                    button
                                    key={index}
                                    style={style}
                                    onClick={() => {
                                        helpers.setValue(timezone);
                                        onChange?.(timezone);
                                        setSearchString('');
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
            {/* Text field that looks like a selector */}
            <TextField
                {...props}
                value={field.value}
                variant="outlined"
                onClick={onOpen}
                InputProps={{
                    endAdornment: (
                        <IconButton size="small" aria-label="timezone-select">
                            {open ?
                                <ArrowDropUpIcon fill={palette.background.textPrimary} /> :
                                <ArrowDropDownIcon fill={palette.background.textPrimary} />
                            }
                        </IconButton>
                    ),
                }}
            />
        </>
    );
};
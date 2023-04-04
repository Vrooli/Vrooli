import { findTimeZone, getUTCOffset, listTimeZones } from 'timezone-support';
import { Selector } from '../Selector/Selector';
import { TimezoneSelectorProps } from '../types';

const formatOffset = (offset) => {
    const sign = offset > 0 ? '-' : '+';
    const hours = Math.abs(Math.floor(offset / 60));
    const minutes = Math.abs(offset % 60);
    return `${sign}${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
};

export const TimezoneSelector = ({
    ...props
}: TimezoneSelectorProps) => {
    const timezones = listTimeZones();
    const currentDate = new Date();

    return (
        <Selector
            {...props}
            options={timezones}
            getOptionLabel={(timezone) => {
                const timeZoneDetails = findTimeZone(timezone);
                const { offset } = getUTCOffset(currentDate, timeZoneDetails);
                const formattedOffset = formatOffset(offset);
                return `${timezone} (UTC ${formattedOffset})`;
            }}
            fullWidth
        />
    );
};
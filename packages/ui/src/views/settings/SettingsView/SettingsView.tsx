import { Box } from '@mui/material';
import { SettingsList, SettingsTopBar } from 'components';
import { SettingsViewProps } from '../types';

export const SettingsView = ({
    display = 'page',
    session,
}: SettingsViewProps) => {
    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Settings',
                }}
            />
            <Box sx={{
                margin: 'auto',
            }}>
                <SettingsList showOnMobile={true} />
            </Box>
        </>
    )
}

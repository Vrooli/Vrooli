import {
    Box,
    IconButton,
    useTheme
} from '@mui/material';
import { CloseIcon } from '@shared/icons';
import { HelpButton } from 'components';
import { noSelect } from 'styles';
import { MenuTitleProps } from '../types';

export const MenuTitle = ({
    ariaLabel,
    helpText,
    onClose,
    title,
}: MenuTitleProps) => {
    const { palette } = useTheme();

    return (
        <Box
            id={ariaLabel}
            sx={{
                ...noSelect,
                display: 'flex',
                alignItems: 'center',
                padding: 2,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                textAlign: 'center',
                fontSize: { xs: '1.5rem', sm: '2rem' },
            }}
        >
            <Box sx={{ marginLeft: 'auto' }} >{title}</Box>
            {helpText && <HelpButton
                markdown={helpText}
                sx={{
                    fill: palette.secondary.light,
                }}
                sxRoot={{
                    display: 'flex',
                    marginTop: 'auto',
                    marginBottom: 'auto',
                }}
            />}
            <IconButton
                aria-label="close"
                edge="end"
                onClick={onClose}
                sx={{ marginLeft: 'auto' }}
            >
                <CloseIcon fill={palette.primary.contrastText} />
            </IconButton>
        </Box>
    )
}
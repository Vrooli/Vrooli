import { useCallback } from 'react';
import { CardGrid, TopBar } from 'components';
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from '@shared/icons';
import { Box, Button, Typography, useTheme } from '@mui/material';
import { CreateViewProps } from '../types';
import { useTranslation } from 'react-i18next';
import { APP_LINKS } from '@shared/consts';
import { useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';

type CreateType = 'Api' | 'Note' | 'Organization' | 'Project' | 'Question' | 'Reminder' | 'Routine' | 'SmartContract' | 'Standard';

type CreateInfo = {
    objectType: CreateType;
    description: CommonKey,
    Icon: SvgComponent,
}

const createCards: CreateInfo[] = [
    {
        objectType: 'Reminder',
        description: 'CreateReminderDescription',
        Icon: ReminderIcon,
    },
    {
        objectType: 'Note',
        description: 'CreateNoteDescription',
        Icon: NoteIcon,
    },
    {
        objectType: 'Routine',
        description: 'CreateRoutineDescription',
        Icon: RoutineIcon,
    },
    {
        objectType: 'Project',
        description: 'CreateProjectDescription',
        Icon: ProjectIcon,
    },
    {
        objectType: 'Organization',
        description: 'CreateOrganizationDescription',
        Icon: OrganizationIcon,
    },
    {
        objectType: 'Question',
        description: 'CreateQuestionDescription',
        Icon: HelpIcon,
    },
    {
        objectType: 'Standard',
        description: 'CreateStandardDescription',
        Icon: StandardIcon,
    },
    {
        objectType: 'SmartContract',
        description: 'CreateSmartContractDescription',
        Icon: SmartContractIcon,
    },
    {
        objectType: 'Api',
        description: 'CreateApiDescription',
        Icon: ApiIcon,
    },
]

export const CreateView = ({
    display = 'page',
    session
}: CreateViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const onSelect = useCallback((objectType: CreateType) => {
        setLocation(`${APP_LINKS[objectType]}/add`);
    }, [setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Create',
                }}
            />
            <CardGrid minWidth={300}>
                {createCards.map(({ objectType, description, Icon }, index) => (
                    <Box
                        key={index}
                        onClick={() => onSelect(objectType)}
                        sx={{
                            width: '100%',
                            boxShadow: 8,
                            padding: 1,
                            borderRadius: 2,
                            cursor: 'pointer',
                            background: palette.background.paper,
                            '&:hover': {
                                filter: 'brightness(1.1)',
                                boxShadow: 12,
                            },
                        }}>
                        {/* Left of card is icon */}
                        <Box sx={{
                            float: 'left',
                            width: '75px',
                            height: '100%',
                            padding: 2,
                        }}>
                            <Box sx={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                width: '100%',
                                height: '100%',
                            }}>
                                <Icon width={'50px'} height={'50px'} fill={palette.background.textPrimary} />
                            </Box>
                        </Box>
                        {/* Right of card is objectType and description */}
                        <Box sx={{
                            float: 'right',
                            width: 'calc(100% - 75px)',
                            height: '100%',
                            padding: '1rem',
                            display: 'contents',
                        }}>
                            <Typography variant='h6' component='div'>
                                {t(objectType, { count: 1 })}
                            </Typography>
                            <Typography variant='body2' color={palette.background.textSecondary}>
                                {t(description)}
                            </Typography>
                            {/* Bottom of card is button */}
                            <Button
                                size='small'
                                sx={{
                                    marginTop: 2,
                                    marginLeft: 'auto',
                                    display: 'flex',
                                }}
                            >{t(`Create`)}</Button>
                        </Box>
                    </Box>
                ))}
            </CardGrid>
        </>
    )
};

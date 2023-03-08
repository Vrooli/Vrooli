import { DialogTitle, Navbar } from 'components';
import { useMemo } from 'react';
import { getTranslatedTitleAndHelp } from 'utils/display';
import { TopBarProps } from '../types';

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = ({
    below,
    display,
    onClose,
    session,
    titleData,
}: TopBarProps) => {
    const { title, help } = useMemo(() => getTranslatedTitleAndHelp(titleData), [titleData]);
    const titleId = useMemo(() => Math.random().toString(36).substring(2, 15), []);

    if (display === 'dialog') return (
        <DialogTitle
            id={titleId}
            title={title}
            onClose={onClose}
        />
    )
    return (
        <Navbar
            below={below}
            help={help}
            session={session}
            shouldHideTitle={titleData?.hideOnDesktop}
            title={title}
        />
    )
};
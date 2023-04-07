import { DialogTitle } from 'components/dialogs/DialogTitle/DialogTitle';
import { useMemo } from 'react';
import { getTranslatedTitleAndHelp } from 'utils/display/translationTools';
import { Navbar } from '../Navbar/Navbar';
import { TopBarProps } from '../types';

/**
 * Generates an app bar for both pages and dialogs
 */
export const TopBar = ({
    below,
    display,
    onClose,
    titleData,
}: TopBarProps) => {
    const { title, help } = useMemo(() => getTranslatedTitleAndHelp(titleData), [titleData]);
    const titleId = useMemo(() => titleData?.titleId ?? Math.random().toString(36).substring(2, 15), [titleData?.titleId]);

    if (display === 'dialog') return (
        <DialogTitle
            below={below}
            id={titleId}
            title={title}
            onClose={onClose}
        />
    )
    return (
        <Navbar
            below={below}
            help={help}
            shouldHideTitle={titleData?.hideOnDesktop}
            title={title}
        />
    )
};
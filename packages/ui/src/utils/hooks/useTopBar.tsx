import { Session } from '@shared/consts';
import { Navbar } from 'components';
import { useMemo } from 'react';
import { OptionalTranslation } from 'types';
import { getTranslatedTitleAndHelp } from 'utils/display';
import { ViewDisplayType } from 'views/types';

type UseNavbarProps = {
    display?: ViewDisplayType;
    session: Session | undefined;
    titleData?: OptionalTranslation & { hideOnDesktop?: boolean; };
    below?: JSX.Element | boolean | undefined;
}

/**
 * Hook for generating a dynamic navbar/appbar for a view
 */
export const useTopBar = ({
    below,
    display,
    session,
    titleData,
}: UseNavbarProps) => {

    const { title, help } = useMemo(() => getTranslatedTitleAndHelp(titleData), [titleData]);
    const TopBar = useMemo(() => (
        <Navbar
            below={below}
            help={help}
            session={session}
            shouldHideTitle={titleData?.hideOnDesktop}
            title={title}
        />
    ), [help, session, title]);

    return TopBar;
};
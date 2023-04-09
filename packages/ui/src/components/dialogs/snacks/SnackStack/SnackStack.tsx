import { Stack } from '@mui/material';
import { uuid } from '@shared/uuid';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getDeviceInfo } from 'utils/display/device';
import { translateSnackMessage } from 'utils/display/translationTools';
import { PubSub, TranslatedSnackMessage, UntranslatedSnackMessage } from 'utils/pubsub';
import { BasicSnack, SnackSeverity } from '../BasicSnack/BasicSnack';
import { CookiesSnack } from '../CookiesSnack/CookiesSnack';
import { BasicSnackProps } from '../types';

/**
 * Displays a stack of snack messages. 
 * Most messages disappear after a few seconds, but some are persistent (e.g. No Internet Connection).
 * No more than 3 ephemeral messages are displayed at once.
 */
export const SnackStack = () => {
    const { t } = useTranslation();

    // FIFO queue of basic snackbars
    const [snacks, setSnacks] = useState<BasicSnackProps[]>([]);
    // Special cookie snack
    const [isCookieSnackOpen, setIsCookieSnackOpen] = useState<boolean>(false);

    // Remove an item from the queue by id
    const handleClose = (id: string) => {
        setSnacks((prev) => [...prev.filter((snack) => snack.id !== id)]);
    };

    // Subscribe to snack events
    useEffect(() => {
        // Subscribe to basic snacks
        let snackSub = PubSub.get().subscribeSnack((o) => {
            // Add the snack to the queue
            setSnacks((snacks) => {
                // event can define an id, or we generate one
                const id = o.id ?? uuid();
                let newSnacks = [...snacks, {
                    buttonClicked: o.buttonClicked,
                    buttonText: o.buttonKey ? t(o.buttonKey, { ...o.buttonVariables, defaultValue: o.buttonKey }) : undefined,
                    data: o.data,
                    handleClose: () => handleClose(id),
                    id,
                    message: (o as UntranslatedSnackMessage).message ?
                        (o as UntranslatedSnackMessage).message :
                        translateSnackMessage((o as TranslatedSnackMessage).messageKey, (o as TranslatedSnackMessage).messageVariables).message,
                    severity: o.severity as SnackSeverity,
                }];
                // Filter out same ids
                newSnacks = newSnacks.filter((snack, index, self) => { return self.findIndex((s) => s.id === snack.id) === index; });
                // Limit the number of snacks
                if (newSnacks.length > 3) {
                    newSnacks = newSnacks.slice(1);
                }
                return newSnacks;
            });
        });
        // Subscribe to special snack events
        let cookiesSub = PubSub.get().subscribeCookies(() => {
            // Ignore if in standalone mode
            if (getDeviceInfo().isStandalone) return;
            setIsCookieSnackOpen(true);
        });
        return () => {
            PubSub.get().unsubscribe(snackSub)
            PubSub.get().unsubscribe(cookiesSub)
        };
    }, [t])

    let visible = useMemo(() => snacks.length > 0 || isCookieSnackOpen, [snacks, isCookieSnackOpen]);

    return (
        // Snacks displayed in bottom left corner
        <Stack direction="column" spacing={1} sx={{
            display: visible ? 'flex' : 'none',
            position: 'fixed',
            // Displays above the bottom nav bar, accounting for PWA inset-bottom
            bottom: { xs: 'calc(64px + env(safe-area-inset-bottom))', md: 'calc(8px + env(safe-area-inset-bottom))' },
            left: 'calc(8px + env(safe-area-inset-left))',
            zIndex: 20000,
            pointerEvents: 'none',
        }}>
            {/* Basic snacks on top */}
            {snacks.map((snack) => (
                <BasicSnack
                    key={snack.id}
                    {...snack}
                />
            ))}
            {/* Special snacks below */}
            {/* Cookie snack */}
            {isCookieSnackOpen && <CookiesSnack handleClose={() => setIsCookieSnackOpen(false)} />}
        </Stack>
    );
}
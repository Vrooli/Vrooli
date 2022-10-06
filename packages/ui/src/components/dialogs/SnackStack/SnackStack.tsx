import { useEffect, useMemo, useState } from 'react';
import { Stack } from '@mui/material';
import { PubSub } from 'utils';
import { Snack } from '../Snack/Snack';
import { SnackProps } from '../types';
import { uuid } from '@shared/uuid';

/**
 * Displays short, temporary messages to the user.
 * Each message is displayed for a few seconds, then disappears.
 * If there are multiple messages, they are displayed in a stack with a maximum number
 */
export const SnackStack = () => {

    // FIFO queue of snackbars
    const [snacks, setSnacks] = useState<SnackProps[]>([]);

    // Remove an item from the queue by id
    const handleClose = (id: string) => {
        setSnacks((prev) => [...prev.filter((snack) => snack.id !== id)]);
    };

    // Subscribe to snack events
    useEffect(() => {
        let snackSub = PubSub.get().subscribeSnack((o) => {
            // Add the snack to the queue
            setSnacks((snacks) => {
                const id = uuid();
                let newSnacks = [...snacks, {
                    buttonClicked: o.buttonClicked,
                    buttonText: o.buttonText,
                    data: o.data,
                    handleClose: () => handleClose(id),
                    id,
                    message: o.message,
                    severity: o.severity,
                }];
                // Limit the number of snacks
                if (newSnacks.length > 3) {
                    newSnacks = newSnacks.slice(1);
                }
                return newSnacks;
            });
        });
        return () => { PubSub.get().unsubscribe(snackSub) };
    }, [])

    let visible = useMemo(() => snacks.length > 0, [snacks]);

    return (
        // Snacks displayed in bottom left corner
        <Stack direction="column" spacing={1} sx={{
            display: visible ? 'block' : 'none',
            position: 'fixed',
            bottom: '8px',
            left: '8px',
            zIndex: 20000,
        }}>
            {snacks.map((snack) => (
                <Snack
                    key={snack.id}
                    {...snack}
                />
            ))}
        </Stack>
    );
}
// Handles wallet integration
/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@local/shared';

export const WALLET_TYPES = {
    Nami: 'nami'
}
export type WALLET_TYPES = ValueOf<typeof WALLET_TYPES>;

export const connectNamiWallet: () => Promise<boolean> = async () => {
    if (!window.cardano) {
        console.error('could not find cardano', window);
        return false;
    }
    try {
        console.log('going to enable...')
        const response = await window.cardano.enable();
        if (response) {
            console.log('enabled!', response);
            return true;
        } else {
            console.log('enable returned false');
            return false;
        }
    } catch (err) {
        // Canceled: code -3
        console.log('enable caught error!', err);
        return false;
    } finally {
        return false;
    }
}

export const connectWallet: () => Promise<boolean> = async (type: WALLET_TYPES = WALLET_TYPES.Nami) => {
    return await connectNamiWallet();
}
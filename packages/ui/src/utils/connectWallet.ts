// Handles wallet integration
/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@local/shared';

export const WALLET_TYPES = {
    Nami: 'nami'
}
export type WALLET_TYPES = ValueOf<typeof WALLET_TYPES>;

export const connectNamiWallet = async (): Promise<boolean> => {
    if (!window.cardano) {
        console.error('could not find cardano', window);
        return false;
    }
    let success = false;
    try {
        const response = await window.cardano.enable();
        if (response) {
            success = true;
        }
    } catch (err) {
        // Canceled: code -3
        console.log('enable caught error!', err);
    } finally {
        return success;
    }
}

export const connectWallet = async (type: WALLET_TYPES = WALLET_TYPES.Nami): Promise<boolean> => {
    return await connectNamiWallet();
}
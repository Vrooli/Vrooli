import { id, name, opt, req, YupModel } from '../utils';
import * as yup from 'yup';

export const walletValidation: YupModel<false, true> = {
    // Cannot create a wallet directly - must go through handshake
    update: () => yup.object().shape({ 
        id: req(id),
        name: opt(name),
    }),
}
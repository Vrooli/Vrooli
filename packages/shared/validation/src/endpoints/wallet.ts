import { id, name, opt, req, YupModel } from '../utils';
import * as yup from 'yup';

export const walletValidation: YupModel = {
    create: yup.object().shape({ }), // Cannot create a wallet directly - must go through handshake
    update: yup.object().shape({ 
        id: req(id),
        name: opt(name),
    }),
}
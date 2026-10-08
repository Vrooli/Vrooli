import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { AccountSecurityService, type AccountSession } from '@vrooli/proto-types/landing-page-business-suite/v1/account_security_pb';
import { CONNECT_API_BASE } from './common';
import { getBrowserBinding } from '../../surfaces/user-auth/lib/browserBinding';

const client = createClient(AccountSecurityService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

const options = () => ({ headers: { 'X-Lpbs-Browser-Binding': getBrowserBinding() } });

export const listAccountSessions = () => client.listSessions({}, options());
export const revokeAccountSession = (sessionId: string) => client.revokeSession({ sessionId }, options());
export const revokeOtherAccountSessions = () => client.revokeOtherSessions({}, options());
export const startAccountReauthentication = () => client.startReauthentication({}, options());
export const completeAccountReauthentication = (code: string, email = '', passkeyAssertion?: Uint8Array, passkeyCeremonyId = '') => client.reauthenticate({ code, email, passkeyAssertion, passkeyCeremonyId }, options());
export const getSignInDeliveryStatus = (email: string) => client.getSignInDeliveryStatus({ email, browserBinding: getBrowserBinding() }, options());
export type { AccountSession };

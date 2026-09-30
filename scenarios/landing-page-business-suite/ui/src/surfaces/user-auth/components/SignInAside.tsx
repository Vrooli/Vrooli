import { KeyRound, MonitorSmartphone, TimerReset } from 'lucide-react';
import { AuthAside } from './AuthAside';

/** Shared explanation beside the customer sign-in card. */
export function SignInAside({ appName }: { appName: string }) {
  return (
    <AuthAside
      eyebrow="Passwordless sign-in"
      title={<>One email.<br /><span>No password.</span></>}
      lede={`New here? The same steps create your ${appName} account.`}
      points={[
        { icon: <KeyRound />, title: 'Nothing to remember', body: 'We email a 6-digit code and a link. There is no password to reuse or leak.' },
        { icon: <TimerReset />, title: 'Short-lived and single-use', body: 'Codes expire in 15 minutes and stop working once you are in.' },
        { icon: <MonitorSmartphone />, title: 'Any inbox, any device', body: 'Type the code here, or open the link on the device you want to use.' },
      ]}
    />
  );
}

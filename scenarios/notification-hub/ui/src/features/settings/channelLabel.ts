import type { TFunction } from "i18next";
import { strings } from "../../consts/strings";

/** A delivery channel as the operator knows it. Unknown keys stay readable. */
export function channelLabel(channel: string, t: TFunction): string {
  switch (channel) {
    case "web_push": return t(strings.settings.channel.webPush);
    case "browser": return t(strings.settings.channel.browser);
    case "push": return t(strings.settings.channel.push);
    case "email": return t(strings.settings.channel.email);
    case "webhook": return t(strings.settings.channel.webhook);
    case "macos_notification": return t(strings.settings.channel.macNotification);
    case "imessage": return t(strings.settings.channel.imessage);
    default: return t(strings.settings.channel.other, { channel });
  }
}

import { Alert } from "@vrooli/react-component-library/Alert/1";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import type { CourierStatus } from "../api/operator";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage, failureKind } from "../lib/errorMessage";
import { deliveryOn } from "../lib/needsYou";

/**
 * Whether what needs the operator reaches them, as one compact line shown once
 * per screen (rulings V3). Silent when delivery works or nothing is open. Open
 * items with nothing being delivered is a warning; `switchAt` links to where
 * delivery is switched.
 */
export function DeliveryBanner({ courier, error, open = 0, switchAt }: { courier?: CourierStatus; error?: unknown; open?: number; switchAt?: string }) {
  const { t } = useTranslation();
  if (error) {
    if (failureKind(error) === "unauthorized") {
      return <Alert tone="danger" title={t(strings.delivery.unauthorizedTitle)} description={t(strings.delivery.unauthorized)} />;
    }
    return <Alert tone="danger" title={t(strings.delivery.unreadableTitle)} description={errorMessage(error, t)} />;
  }
  if (!courier) return null;
  const action = switchAt ? <Link to={switchAt} className="rd-more">{t(strings.delivery.switch)}</Link> : undefined;
  if (!courier.writer) {
    return <Alert tone="warning" title={t(strings.delivery.notWriterTitle)} description={courier.writerReason || t(strings.delivery.notWriter)} />;
  }
  if (open === 0) return null;
  if (!deliveryOn(courier)) return <WarningLine text={t(strings.delivery.allOff, { count: open })} action={action} />;
  if (!courier.hubRead) return <WarningLine text={`${t(strings.delivery.hubUnreadTitle)}: ${courier.hubReason || t(strings.common.unknown)}`} action={action} />;
  return null;
}

/** A warning that costs one line of a phone screen, not a card (rulings V3). */
function WarningLine({ text, action }: { text: string; action?: ReactNode }) {
  return (
    <p role="status" data-tone="warning" className="rd-warn-line">
      <span>{text}</span>
      {action}
    </p>
  );
}

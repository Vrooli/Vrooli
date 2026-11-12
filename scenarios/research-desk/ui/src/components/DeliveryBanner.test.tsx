import { Code, ConnectError } from "@connectrpc/connect";
import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import type { CourierStatus } from "../api/operator";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { DeliveryBanner } from "./DeliveryBanner";

afterEach(() => cleanup());

const courier = (fields: Partial<CourierStatus>) => ({ writer: true, globalOn: true, hubRead: true, commissions: [], ...fields }) as CourierStatus;

describe("DeliveryBanner", () => {
  it.each([
    ["not authorized", { error: new ConnectError("no", Code.PermissionDenied) }, strings.delivery.unauthorizedTitle],
    ["unreadable", { error: new TypeError("Failed to fetch") }, strings.delivery.unreadableTitle],
    ["another desk delivers", { courier: courier({ writer: false }) }, strings.delivery.notWriterTitle],
    ["items are open and delivery is off everywhere", { courier: courier({ globalOn: false }), open: 7 }, strings.delivery.allOff],
    ["Notification Hub unread", { courier: courier({ hubRead: false, hubReason: "hub down" }), open: 1 }, strings.delivery.hubUnreadTitle],
  ])("says plainly when %s", (_name, props, title) => {
    renderWithProviders(<DeliveryBanner {...props} />);
    expect(screen.getByText(new RegExp(title))).toBeInTheDocument();
  });

  it("warns in one line, linking to where delivery is switched, when open items reach no phone", () => {
    renderWithProviders(<DeliveryBanner courier={courier({ globalOn: false })} open={7} switchAt="/needs-you#delivery" />);
    expect(screen.getByRole("status")).toHaveAttribute("data-tone", "warning");
    expect(screen.getByRole("link", { name: strings.delivery.switch })).toHaveAttribute("href", "/needs-you#delivery");
  });

  it.each([
    ["delivery works", { courier: courier({}), open: 3 }],
    ["nothing is open, even with delivery off", { courier: courier({ globalOn: false }), open: 0 }],
  ])("stays silent when %s", (_name, props) => {
    const { container } = renderWithProviders(<DeliveryBanner {...props} />);
    expect(container.textContent).toBe("");
  });
});

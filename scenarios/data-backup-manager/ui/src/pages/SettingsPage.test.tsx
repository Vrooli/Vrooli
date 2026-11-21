import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { SettingsPage } from "./SettingsPage";

vi.mock("@vrooli/react-component-library/AppNotificationSettings/1", () => ({
	AppNotificationSettings: () => <div data-testid="app-notification-settings" />,
}));

afterEach(() => cleanup());

describe("SettingsPage", () => {
	it("shows the shared app-delivery settings panel", () => {
		renderWithProviders(<SettingsPage />);

		expect(screen.getByRole("heading", { name: strings.pages.settings.notificationsHeading })).toBeInTheDocument();
		expect(screen.getByTestId("app-notification-settings")).toBeInTheDocument();
	});
});

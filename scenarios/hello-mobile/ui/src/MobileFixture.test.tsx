import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import App from "./App";
import { selectors } from "./consts/selectors";

describe("hello-mobile fixture", () => {
	it("derives a deterministic response and persists the input state", () => {
		localStorage.clear();
		render(<App />);
		fireEvent.change(screen.getByTestId(selectors.helloMobile.input), { target: { value: "android" } });
		fireEvent.click(screen.getByTestId(selectors.helloMobile.submit));
		expect(screen.getByTestId(selectors.helloMobile.result)).toHaveTextContent("Result: DIORDNA");
		expect(screen.getByTestId(selectors.helloMobile.state)).toHaveTextContent("Saved state: android");
		expect(localStorage.getItem("hello-mobile-state")).toBe("android");
	});

	it("exposes explicit connectivity and notification targets", () => {
		render(<App />);
		fireEvent.click(screen.getByRole("button", { name: "Toggle connectivity" }));
		expect(screen.getByTestId(selectors.helloMobile.connectivity)).toHaveTextContent("Connectivity: offline");
		fireEvent.click(screen.getByTestId(selectors.helloMobile.notification));
		expect(screen.getByTestId(selectors.helloMobile.result)).toHaveTextContent("NOTIFICATION_OPENED");
	});
});

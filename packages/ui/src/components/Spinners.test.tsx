import React from "react";
import { describe, test, expect, vi } from "vitest";
import { render, screen } from "../__test/testUtils.js";
import { DiagonalWaveLoader } from "./Spinners.js";

// Mock the useMenu hook to prevent issues
vi.mock("../hooks/useMenu.js", () => ({
    useMenu: vi.fn(() => ({
        anchorEl: null,
        open: false,
        openMenu: vi.fn(),
        closeMenu: vi.fn(),
    })),
}));

describe("<DiagonalWaveLoader />", () => {
    // Test 1: Renders without crashing
    test("renders without crashing", () => {
        render(<DiagonalWaveLoader data-testid="diagonal-wave-loader" />);
        const loader = screen.getByTestId("diagonal-wave-loader");
        expect(loader).toBeDefined();
    });
});


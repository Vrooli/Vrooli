import React from "react";
import { describe, test, expect } from "vitest";
import { render, screen } from "../__test/testUtils.js";
import { DiagonalWaveLoader } from "./Spinners.js";

describe("<DiagonalWaveLoader />", () => {
    // Test 1: Renders without crashing
    test("renders without crashing", () => {
        render(<DiagonalWaveLoader data-testid="diagonal-wave-loader" />);
        const loader = screen.getByTestId("diagonal-wave-loader");
        expect(loader).toBeDefined();
    });
});


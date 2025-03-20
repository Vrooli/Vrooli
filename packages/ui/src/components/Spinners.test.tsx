import { expect } from "chai";
import React from "react";
import { render, screen } from "../../__mocks__/testUtils.js";
import { DiagonalWaveLoader } from "./Spinners.js";

describe("<DiagonalWaveLoader />", () => {
    // Test 1: Renders without crashing
    test("renders without crashing", () => {
        render(<DiagonalWaveLoader data-testid="diagonal-wave-loader" />);
        const loader = screen.getByTestId("diagonal-wave-loader");
        expect(loader).toBeInTheDocument();
    });
});


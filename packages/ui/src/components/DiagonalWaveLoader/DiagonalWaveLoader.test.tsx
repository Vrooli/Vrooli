import { render, screen } from "@testing-library/react";
import { DiagonalWaveLoader } from "./DiagonalWaveLoader";

describe("<DiagonalWaveLoader />", () => {

    // Test 1: Renders without crashing
    test("renders without crashing", () => {
        render(<DiagonalWaveLoader data-testid="diagonal-wave-loader" />);
        const loader = screen.getByTestId("diagonal-wave-loader");
        expect(loader).toBeInTheDocument();
    });

});


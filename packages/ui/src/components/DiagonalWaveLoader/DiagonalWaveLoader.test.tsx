import { render, screen } from "../../__mocks__/testUtils";
import { DiagonalWaveLoader } from "./DiagonalWaveLoader";

describe("<DiagonalWaveLoader />", () => {

    // Test 1: Renders without crashing
    test("renders without crashing", () => {
        render(<DiagonalWaveLoader data-testid="diagonal-wave-loader" />);
        const loader = screen.getByTestId("diagonal-wave-loader");
        expect(loader).toBeInTheDocument();
    });

});


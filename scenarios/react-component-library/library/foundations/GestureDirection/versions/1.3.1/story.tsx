import {
  resolveAnchorEdge,
  resolveGestureDirection,
  type AnchorEdge,
  type WritingDirection,
} from "./GestureDirection";

const DIRECTIONS: WritingDirection[] = ["ltr", "rtl"];
const ANCHORS: AnchorEdge[] = ["inline-start", "inline-end"];

/**
 * The whole truth table, rendered. Two independent inputs, four physical
 * outcomes, and the pair that reaches the same side by opposite routes is the
 * pair a single-input implementation collapses by accident.
 */
export function Default() {
  return (
    <table data-testid="foundations.gesture-direction">
      <thead>
        <tr>
          <th scope="col">Writing direction</th>
          <th scope="col">Anchor</th>
          <th scope="col">Rests against</th>
          <th scope="col">Dismiss</th>
          <th scope="col">Reveal</th>
        </tr>
      </thead>
      <tbody>
        {DIRECTIONS.flatMap((direction) =>
          ANCHORS.map((anchor) => (
            <tr key={`${direction}-${anchor}`} data-testid={`${direction}-${anchor}`}>
              <td>{direction}</td>
              <td>{anchor}</td>
              <td data-anchor-edge={resolveAnchorEdge(direction, anchor)}>
                {resolveAnchorEdge(direction, anchor)}
              </td>
              <td>{resolveGestureDirection(direction, anchor, "dismiss")}</td>
              <td>{resolveGestureDirection(direction, anchor, "reveal")}</td>
            </tr>
          )),
        )}
      </tbody>
    </table>
  );
}

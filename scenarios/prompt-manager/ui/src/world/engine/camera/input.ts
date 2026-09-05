import CameraControls from 'camera-controls'
import type { CameraTuning } from '../../config'

type InputMap = CameraTuning['input']
const A = CameraControls.ACTION
const mouse = {
  none: A.NONE, rotate: A.ROTATE, truck: A.TRUCK, offset: A.OFFSET, dolly: A.DOLLY, zoom: A.ZOOM,
} satisfies Record<InputMap['mouse']['left'], number>
const single = {
  none: A.NONE, rotate: A.TOUCH_ROTATE, truck: A.TOUCH_TRUCK, 'screen-pan': A.TOUCH_SCREEN_PAN,
  offset: A.TOUCH_OFFSET, dolly: A.DOLLY, zoom: A.ZOOM,
} satisfies Record<InputMap['touch']['one'], number>
const multi = {
  none: A.NONE, rotate: A.TOUCH_ROTATE, truck: A.TOUCH_TRUCK, 'screen-pan': A.TOUCH_SCREEN_PAN,
  offset: A.TOUCH_OFFSET, dolly: A.TOUCH_DOLLY, zoom: A.TOUCH_ZOOM,
  'dolly-truck': A.TOUCH_DOLLY_TRUCK, 'dolly-screen-pan': A.TOUCH_DOLLY_SCREEN_PAN,
  'dolly-offset': A.TOUCH_DOLLY_OFFSET, 'dolly-rotate': A.TOUCH_DOLLY_ROTATE,
  'zoom-truck': A.TOUCH_ZOOM_TRUCK, 'zoom-screen-pan': A.TOUCH_ZOOM_SCREEN_PAN,
  'zoom-offset': A.TOUCH_ZOOM_OFFSET, 'zoom-rotate': A.TOUCH_ZOOM_ROTATE,
} satisfies Record<InputMap['touch']['two'], number>

function action(map: Readonly<Record<string, number>>, value: string): number {
  if (!Object.prototype.hasOwnProperty.call(map, value)) throw new Error(`Unmapped camera input action: ${value}`)
  const result = map[value]
  if (result === undefined) throw new Error(`Undefined camera input action: ${value}`)
  return result
}

/** Structural numeric slots include runtime actions omitted by upstream typings. */
interface InputControls {
  mouseButtons: Record<keyof InputMap['mouse'], number>
  touches: Record<keyof InputMap['touch'], number>
}

/** Resolve all seven slots before mutating controls; invalid maps never partially apply. */
export function applyInputMap(controls: InputControls, input: InputMap): void {
  const mouseButtons = {
    left: action(mouse, input.mouse.left), middle: action(mouse, input.mouse.middle),
    right: action(mouse, input.mouse.right), wheel: action(mouse, input.mouse.wheel),
  }
  const touches = {
    one: action(single, input.touch.one), two: action(multi, input.touch.two), three: action(multi, input.touch.three),
  }
  controls.mouseButtons = mouseButtons
  controls.touches = touches
}

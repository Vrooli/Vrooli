export interface ImageView { scale: number; x: number; y: number }
export interface ImageDimensions { width: number; height: number }

export function rotatedDimensions(image: ImageDimensions, rotation: number): ImageDimensions {
  return rotation % 180 === 0 ? image : { width: image.height, height: image.width };
}

export function fitImage(image: ImageDimensions, stage: ImageDimensions): number {
  if (image.width <= 0 || image.height <= 0 || stage.width <= 0 || stage.height <= 0) return 1;
  return Math.min(1, stage.width / image.width, stage.height / image.height);
}

export function boundImage(view: ImageView, image: ImageDimensions, stage: ImageDimensions): ImageView {
  const maxX = Math.max(0, (image.width * view.scale - stage.width) / 2);
  const maxY = Math.max(0, (image.height * view.scale - stage.height) / 2);
  return { scale: view.scale, x: maxX === 0 ? 0 : Math.max(-maxX, Math.min(maxX, view.x)), y: maxY === 0 ? 0 : Math.max(-maxY, Math.min(maxY, view.y)) };
}

export function zoomImage(view: ImageView, scale: number, at: { x: number; y: number }, image: ImageDimensions, stage: ImageDimensions): ImageView {
  const next = Math.max(0.01, Math.min(8, scale));
  const ratio = next / view.scale;
  return boundImage({ scale: next, x: at.x - (at.x - view.x) * ratio, y: at.y - (at.y - view.y) * ratio }, image, stage);
}

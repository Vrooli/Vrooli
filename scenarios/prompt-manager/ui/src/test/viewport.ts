interface ViewportSnapshot {
  innerWidth: number
  innerHeight: number
}

const defaultViewport: ViewportSnapshot = {
  innerWidth: window.innerWidth,
  innerHeight: window.innerHeight,
}

export function setViewport(width: number, height = window.innerHeight) {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    writable: true,
    value: width,
  })
  Object.defineProperty(window, 'innerHeight', {
    configurable: true,
    writable: true,
    value: height,
  })
  window.dispatchEvent(new Event('resize'))
}

export function restoreViewport() {
  setViewport(defaultViewport.innerWidth, defaultViewport.innerHeight)
}

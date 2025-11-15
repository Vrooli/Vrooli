import { useCallback, useEffect, useState } from 'react'

const FULLSCREEN_EVENTS = ['fullscreenchange', 'webkitfullscreenchange', 'mozfullscreenchange', 'MSFullscreenChange'] as const

const useFullscreenManager = (canvasRef: React.RefObject<HTMLDivElement>) => {
  const [canFullscreen, setCanFullscreen] = useState(true)
  const [isNativeFullscreen, setIsNativeFullscreen] = useState(false)
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false)

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    const fullscreenSupported = Boolean(
      document.fullscreenEnabled ||
        (document as typeof document & { webkitFullscreenEnabled?: boolean }).webkitFullscreenEnabled ||
        (document as typeof document & { mozFullScreenEnabled?: boolean }).mozFullScreenEnabled ||
        (document as typeof document & { msFullscreenEnabled?: boolean }).msFullscreenEnabled ||
        document.fullscreenEnabled === undefined
    )

    setCanFullscreen(fullscreenSupported || true)

    const handleFullscreenChange = () => {
      const fullscreenElement =
        document.fullscreenElement ||
        (document as typeof document & { webkitFullscreenElement?: Element | null }).webkitFullscreenElement ||
        (document as typeof document & { mozFullScreenElement?: Element | null }).mozFullScreenElement ||
        (document as typeof document & { msFullscreenElement?: Element | null }).msFullscreenElement

      const canvasElement = canvasRef.current
      const isCanvasFullscreen = Boolean(fullscreenElement && canvasElement && canvasElement.contains(fullscreenElement))

      setIsNativeFullscreen(isCanvasFullscreen)

      if (fullscreenElement && canvasElement && !canvasElement.contains(fullscreenElement)) {
        setIsLayoutFullscreen(false)
      }
      if (!fullscreenElement) {
        setIsNativeFullscreen(false)
      }
    }

    FULLSCREEN_EVENTS.forEach((event) => document.addEventListener(event, handleFullscreenChange))
    handleFullscreenChange()

    return () => {
      FULLSCREEN_EVENTS.forEach((event) => document.removeEventListener(event, handleFullscreenChange))
    }
  }, [canvasRef])

  const toggleFullscreen = useCallback(() => {
    const element = canvasRef.current
    if (!element) {
      return
    }

    if (!canFullscreen) {
      setIsLayoutFullscreen((previous) => !previous)
      return
    }

    if (isNativeFullscreen || isLayoutFullscreen) {
      const exitFullscreen =
        document.exitFullscreen ||
        (document as typeof document & { webkitExitFullscreen?: () => Promise<void> }).webkitExitFullscreen ||
        (document as typeof document & { mozCancelFullScreen?: () => Promise<void> }).mozCancelFullScreen ||
        (document as typeof document & { msExitFullscreen?: () => Promise<void> }).msExitFullscreen
      if (typeof exitFullscreen === 'function') {
        exitFullscreen.call(document)
      } else {
        setIsLayoutFullscreen(false)
      }
      return
    }

    const requestFullscreen =
      element.requestFullscreen ||
      (element as typeof element & { webkitRequestFullscreen?: () => Promise<void> }).webkitRequestFullscreen ||
      (element as typeof element & { mozRequestFullScreen?: () => Promise<void> }).mozRequestFullScreen ||
      (element as typeof element & { msRequestFullscreen?: () => Promise<void> }).msRequestFullscreen

    if (typeof requestFullscreen === 'function') {
      const enterNativeFullscreen = async () => {
        const fullscreenElement =
          document.fullscreenElement ||
          (document as typeof document & { webkitFullscreenElement?: Element | null }).webkitFullscreenElement ||
          (document as typeof document & { mozFullScreenElement?: Element | null }).mozFullScreenElement ||
          (document as typeof document & { msFullscreenElement?: Element | null }).msFullscreenElement

        if (
          fullscreenElement &&
          fullscreenElement !== element &&
          typeof document.exitFullscreen === 'function'
        ) {
          await document.exitFullscreen()
        }

        await requestFullscreen.call(element)
      }

      enterNativeFullscreen().catch(() => {
        setIsLayoutFullscreen(true)
      })
    } else {
      setIsLayoutFullscreen((previous) => !previous)
    }
  }, [canvasRef, canFullscreen, isLayoutFullscreen, isNativeFullscreen])

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    const immersiveClass = 'tech-tree-immersive-active'
    const { body } = document

    if (isLayoutFullscreen) {
      body.classList.add(immersiveClass)
    } else {
      body.classList.remove(immersiveClass)
    }

    return () => {
      body.classList.remove(immersiveClass)
    }
  }, [isLayoutFullscreen])

  useEffect(() => {
    if (!isLayoutFullscreen || typeof window === 'undefined') {
      return undefined
    }

    const handleKeydown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' || event.key === 'Esc') {
        setIsLayoutFullscreen(false)
      }
    }

    window.addEventListener('keydown', handleKeydown)

    return () => {
      window.removeEventListener('keydown', handleKeydown)
    }
  }, [isLayoutFullscreen])

  return {
    canFullscreen,
    isFullscreen: isNativeFullscreen || isLayoutFullscreen,
    isLayoutFullscreen,
    toggleFullscreen
  }
}

export default useFullscreenManager

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

interface UseVirtualRowsOptions {
  count: number
  rowHeight: number
  overscan?: number
}

interface VirtualRow {
  index: number
  offsetTop: number
}

export function useVirtualRows({
  count,
  rowHeight,
  overscan = 6,
}: UseVirtualRowsOptions) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [viewport, setViewport] = useState({ scrollTop: 0, height: 0 })

  const refreshViewport = useCallback(() => {
    const element = containerRef.current
    if (!element) return
    setViewport({
      scrollTop: element.scrollTop,
      height: element.clientHeight,
    })
  }, [])

  useEffect(() => {
    const element = containerRef.current
    if (!element) return

    refreshViewport()
    element.addEventListener('scroll', refreshViewport, { passive: true })

    const observer = typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(refreshViewport)
      : null
    observer?.observe(element)

    return () => {
      element.removeEventListener('scroll', refreshViewport)
      observer?.disconnect()
    }
  }, [refreshViewport])

  const totalHeight = count * rowHeight
  const startIndex = Math.max(0, Math.floor(viewport.scrollTop / rowHeight) - overscan)
  const visibleCount = Math.ceil((viewport.height || rowHeight) / rowHeight) + overscan * 2
  const endIndex = Math.min(count, startIndex + visibleCount)

  const virtualRows = useMemo<VirtualRow[]>(() => {
    const rows: VirtualRow[] = []
    for (let index = startIndex; index < endIndex; index++) {
      rows.push({ index, offsetTop: index * rowHeight })
    }
    return rows
  }, [endIndex, rowHeight, startIndex])

  return {
    containerRef,
    totalHeight,
    virtualRows,
  }
}

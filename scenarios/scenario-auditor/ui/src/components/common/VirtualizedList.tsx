import { CSSProperties, ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import clsx from 'clsx'

type VirtualizedListProps<T> = {
  items: readonly T[]
  renderItem: (item: T, index: number) => ReactNode
  getItemKey?: (item: T, index: number) => string | number
  estimatedItemHeight?: number
  overscan?: number
  className?: string
  style?: CSSProperties
  emptyPlaceholder?: ReactNode
}

type ResizeMap = Map<number, ResizeObserver>

const DEFAULT_ESTIMATE = 240

export function VirtualizedList<T>({
  items,
  renderItem,
  getItemKey,
  estimatedItemHeight = DEFAULT_ESTIMATE,
  overscan = 4,
  className,
  style,
  emptyPlaceholder,
}: VirtualizedListProps<T>) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const observersRef = useRef<ResizeMap>(new Map())
  const nodesRef = useRef<Map<number, HTMLDivElement | null>>(new Map())
  const rafRef = useRef<Map<number, number>>(new Map())

  const [scrollTop, setScrollTop] = useState(0)
  const [viewportHeight, setViewportHeight] = useState(0)
  const [heights, setHeights] = useState<number[]>(() =>
    items.map(() => estimatedItemHeight)
  )

  useEffect(() => {
    setHeights(prev => {
      if (prev.length === items.length) {
        return prev
      }
      const next = items.map((_, index) => prev[index] ?? estimatedItemHeight)
      return next
    })
  }, [items, estimatedItemHeight])

  useEffect(() => {
    const container = containerRef.current
    if (!container) {
      return
    }

    const handleScroll = () => {
      setScrollTop(container.scrollTop)
    }

    const resizeObserver = new ResizeObserver(entries => {
      const entry = entries[0]
      if (entry) {
        setViewportHeight(entry.contentRect.height)
      }
    })

    container.addEventListener('scroll', handleScroll)
    resizeObserver.observe(container)
    setViewportHeight(container.clientHeight)
    setScrollTop(container.scrollTop)

    return () => {
      container.removeEventListener('scroll', handleScroll)
      resizeObserver.disconnect()
    }
  }, [items.length])

  useEffect(() => {
    return () => {
      observersRef.current.forEach(observer => observer.disconnect())
      observersRef.current.clear()
      nodesRef.current.clear()
      rafRef.current.forEach(id => cancelAnimationFrame(id))
      rafRef.current.clear()
    }
  }, [])

  const cumulativeHeights = useMemo(() => {
    const offsets = new Array(heights.length + 1)
    offsets[0] = 0
    for (let i = 0; i < heights.length; i += 1) {
      offsets[i + 1] = offsets[i] + (Number.isFinite(heights[i]) ? heights[i] : estimatedItemHeight)
    }
    return offsets
  }, [heights, estimatedItemHeight])

  const totalHeight = cumulativeHeights[cumulativeHeights.length - 1] ?? 0

  useEffect(() => {
    const container = containerRef.current
    if (!container) {
      return
    }

    const maxScroll = Math.max(0, totalHeight - container.clientHeight)
    if (container.scrollTop > maxScroll) {
      container.scrollTop = maxScroll
    }
  }, [totalHeight, items.length])

  const findStartIndex = useCallback(
    (offset: number) => {
      let low = 0
      let high = cumulativeHeights.length - 1
      while (low < high) {
        const mid = Math.floor((low + high) / 2)
        if (cumulativeHeights[mid] <= offset) {
          low = mid + 1
        } else {
          high = mid
        }
      }
      return Math.max(0, low - 1)
    },
    [cumulativeHeights]
  )

  const startIndex = items.length === 0 ? 0 : Math.max(0, findStartIndex(scrollTop) - overscan)
  const viewportEnd = scrollTop + viewportHeight
  const endIndex = items.length === 0
    ? 0
    : Math.min(items.length, findStartIndex(viewportEnd) + 1 + overscan)

  const setMeasuredNode = useCallback((index: number, node: HTMLDivElement | null) => {
    const prevNode = nodesRef.current.get(index)
    if (prevNode === node) {
      return
    }

    const existingObserver = observersRef.current.get(index)
    if (existingObserver) {
      existingObserver.disconnect()
      observersRef.current.delete(index)
    }

    const scheduled = rafRef.current.get(index)
    if (scheduled !== undefined) {
      cancelAnimationFrame(scheduled)
      rafRef.current.delete(index)
    }

    if (node) {
      nodesRef.current.set(index, node)

      const measure = () => {
        const rect = node.getBoundingClientRect()
        const height = rect.height
        setHeights(prev => {
          const current = prev[index]

          if (!Number.isFinite(height)) {
            return prev
          }

          if (height <= 0 && current !== undefined) {
            return prev
          }

          if (current !== undefined && Math.abs(current - height) < 0.5) {
            return prev
          }

          const next = [...prev]
          next[index] = height > 0 ? height : estimatedItemHeight
          return next
        })
      }

      const scheduleMeasure = () => {
        if (rafRef.current.has(index)) {
          return
        }
        const id = requestAnimationFrame(() => {
          rafRef.current.delete(index)
          measure()
        })
        rafRef.current.set(index, id)
      }

      const resizeObserver = new ResizeObserver(() => scheduleMeasure())
      resizeObserver.observe(node)
      observersRef.current.set(index, resizeObserver)
      scheduleMeasure()
    } else {
      nodesRef.current.delete(index)
    }
  }, [])

  const containerClasses = clsx('overflow-y-auto', className)
  const containerStyle: CSSProperties = {
    position: 'relative',
    ...style,
  }

  if (items.length === 0) {
    return (
      <div ref={containerRef} className={containerClasses} style={containerStyle}>
        {emptyPlaceholder ?? null}
      </div>
    )
  }

  return (
    <div ref={containerRef} className={containerClasses} style={containerStyle}>
      <div style={{ height: totalHeight, position: 'relative' }}>
        {items.slice(startIndex, endIndex).map((item, localIdx) => {
          const index = startIndex + localIdx
          const top = cumulativeHeights[index]
          const key = getItemKey ? getItemKey(item, index) : index

          return (
            <div
              key={key}
              ref={node => setMeasuredNode(index, node)}
              style={{
                position: 'absolute',
                top,
                left: 0,
                right: 0,
                width: '100%',
              }}
            >
              {renderItem(item, index)}
            </div>
          )
        })}
      </div>
    </div>
  )
}

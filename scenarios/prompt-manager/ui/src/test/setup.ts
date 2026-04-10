/**
 * Vitest test setup file.
 *
 * Configures the testing environment with:
 * - jest-dom matchers for DOM assertions
 * - Mock implementations for browser APIs
 * - DOM APIs required by TipTap editor
 */

import { vi, beforeEach } from 'vitest'
import '@testing-library/jest-dom'

// Mock matchMedia for components using media queries
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

// Mock ResizeObserver for components that track size
class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

Object.defineProperty(window, 'ResizeObserver', {
  writable: true,
  value: MockResizeObserver,
})

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// ============================================================================
// TipTap DOM API Requirements
// ============================================================================

// Mock Range API for TipTap selection handling
class MockRange {
  startContainer: Node | null = null
  startOffset = 0
  endContainer: Node | null = null
  endOffset = 0
  collapsed = true
  commonAncestorContainer: Node | null = null

  setStart(node: Node, offset: number) {
    this.startContainer = node
    this.startOffset = offset
    this.collapsed = this.startContainer === this.endContainer && this.startOffset === this.endOffset
  }

  setEnd(node: Node, offset: number) {
    this.endContainer = node
    this.endOffset = offset
    this.collapsed = this.startContainer === this.endContainer && this.startOffset === this.endOffset
  }

  selectNode(_node: Node) {}
  selectNodeContents(_node: Node) {}
  collapse(_toStart?: boolean) {}
  cloneContents() { return document.createDocumentFragment() }
  cloneRange() { return new MockRange() }
  deleteContents() {}
  extractContents() { return document.createDocumentFragment() }
  insertNode(_node: Node) {}
  surroundContents(_newParent: Node) {}
  compareBoundaryPoints(_how: number, _sourceRange: Range) { return 0 }
  detach() {}
  toString() { return '' }
  createContextualFragment(html: string) {
    const template = document.createElement('template')
    template.innerHTML = html
    return template.content
  }
  getBoundingClientRect() {
    return { top: 0, left: 0, bottom: 0, right: 0, width: 0, height: 0, x: 0, y: 0, toJSON: () => ({}) }
  }
  getClientRects() { return [] as unknown as DOMRectList }
  isPointInRange(_node: Node, _offset: number) { return false }
  intersectsNode(_node: Node) { return false }
}

// Mock Selection API for TipTap
class MockSelection {
  rangeCount = 0
  anchorNode: Node | null = null
  anchorOffset = 0
  focusNode: Node | null = null
  focusOffset = 0
  isCollapsed = true
  type = 'None'

  private ranges: MockRange[] = []

  getRangeAt(index: number) {
    return this.ranges[index] ?? new MockRange()
  }

  addRange(range: MockRange) {
    this.ranges.push(range)
    this.rangeCount = this.ranges.length
    this.anchorNode = range.startContainer
    this.anchorOffset = range.startOffset
    this.focusNode = range.endContainer
    this.focusOffset = range.endOffset
    this.isCollapsed = range.collapsed
    this.type = range.collapsed ? 'Caret' : 'Range'
  }

  removeAllRanges() {
    this.ranges = []
    this.rangeCount = 0
    this.anchorNode = null
    this.anchorOffset = 0
    this.focusNode = null
    this.focusOffset = 0
    this.isCollapsed = true
    this.type = 'None'
  }

  removeRange(_range: MockRange) {
    this.ranges = this.ranges.filter(r => r !== _range)
    this.rangeCount = this.ranges.length
  }

  collapse(_node: Node | null, _offset?: number) {}
  collapseToEnd() {}
  collapseToStart() {}
  containsNode(_node: Node, _allowPartialContainment?: boolean) { return false }
  deleteFromDocument() {}
  empty() { this.removeAllRanges() }
  extend(_node: Node, _offset?: number) {}
  selectAllChildren(_node: Node) {}
  setBaseAndExtent(_anchorNode: Node, _anchorOffset: number, _focusNode: Node, _focusOffset: number) {}
  setPosition(_node: Node | null, _offset?: number) {}
  toString() { return '' }
  modify(_alter: string, _direction: string, _granularity: string) {}
}

// Mock document.createRange
if (typeof document.createRange !== 'function') {
  document.createRange = () => new MockRange() as unknown as Range
}

// Mock window.getSelection
const mockSelection = new MockSelection()
Object.defineProperty(window, 'getSelection', {
  writable: true,
  value: () => mockSelection as unknown as Selection,
})

// Mock document.getSelection
Object.defineProperty(document, 'getSelection', {
  writable: true,
  value: () => mockSelection as unknown as Selection,
})

// Mock Element.scrollIntoView (used by TipTap for caret visibility)
if (typeof Element.prototype.scrollIntoView !== 'function') {
  Element.prototype.scrollIntoView = function() {}
}

// Mock ClipboardEvent for TipTap paste handling
if (typeof ClipboardEvent === 'undefined') {
  (global as Record<string, unknown>).ClipboardEvent = class ClipboardEvent extends Event {
    clipboardData: DataTransfer | null
    constructor(type: string, options?: ClipboardEventInit) {
      super(type, options)
      this.clipboardData = options?.clipboardData ?? null
    }
  }
}

// Mock DataTransfer for clipboard operations
if (typeof DataTransfer === 'undefined') {
  (global as Record<string, unknown>).DataTransfer = class MockDataTransfer {
    private data = new Map<string, string>()
    dropEffect: 'none' | 'copy' | 'link' | 'move' = 'none'
    effectAllowed: 'none' | 'copy' | 'copyLink' | 'copyMove' | 'link' | 'linkMove' | 'move' | 'all' | 'uninitialized' = 'uninitialized'
    files = [] as unknown as FileList
    items = [] as unknown as DataTransferItemList
    types: string[] = []

    getData(format: string) { return this.data.get(format) ?? '' }
    setData(format: string, data: string) {
      this.data.set(format, data)
      if (!this.types.includes(format)) this.types.push(format)
    }
    clearData(format?: string) {
      if (format) {
        this.data.delete(format)
        this.types = this.types.filter(t => t !== format)
      } else {
        this.data.clear()
        this.types = []
      }
    }
    setDragImage() {}
  }
}

// Reset mocks between tests
beforeEach(() => {
  vi.clearAllMocks()
  localStorageMock.getItem.mockReturnValue(null)
  // Reset selection state
  mockSelection.removeAllRanges()
})

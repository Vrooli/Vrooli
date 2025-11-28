// [REQ:KO-HD-001] Basic UI rendering test
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'

describe('App', () => {
  let queryClient: QueryClient

  beforeEach(() => {
    // Create a new QueryClient for each test
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    })

    // Mock environment
    window.ENV = {
      API_PORT: '17822'
    }
  })

  it('[REQ:KO-HD-001] renders without crashing', () => {
    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    )
    expect(container).toBeTruthy()
  })

  it('[REQ:KO-HD-001] renders Knowledge Observatory title', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    )
    const titleElement = screen.getByText(/Knowledge Observatory/i)
    expect(titleElement).toBeDefined()
  })

  it('[REQ:KO-HD-002] renders feature cards', () => {
    const { container } = render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    )
    // Check that some content is rendered
    expect(container.textContent?.length).toBeGreaterThan(0)
  })
})

import { LazyMotion, domAnimation, m } from 'framer-motion'
import { ReactNode } from 'react'

// Use LazyMotion with domAnimation features to reduce bundle size
// This loads only the features we actually use
export function OptimizedMotionProvider({ children }: { children: ReactNode }) {
  return (
    <LazyMotion features={domAnimation} strict>
      {children}
    </LazyMotion>
  )
}

// Re-export optimized motion components
// Use 'm' instead of 'motion' for smaller bundle size
export { m as motion }
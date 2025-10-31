import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { Funnel } from '../types'
import { loadFunnelFromCache } from '../utils/funnelCache'

const FunnelPreview = () => {
  const { id } = useParams()
  const [funnel, setFunnel] = useState<Funnel | null>(null)
  const [currentStepIndex, setCurrentStepIndex] = useState(0)
  const [responses, setResponses] = useState<Record<string, any>>({})
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!id) {
      setFunnel(null)
      setIsLoading(false)
      return
    }

    setIsLoading(true)
    const cachedFunnel = loadFunnelFromCache(id)
    setFunnel(cachedFunnel)
    setIsLoading(false)
  }, [id])

  useEffect(() => {
    if (typeof window === 'undefined' || !id) {
      return
    }

    const handleStorage = (event: StorageEvent) => {
      if (event.key === `funnel-preview:${id}`) {
        if (!event.newValue) {
          setFunnel(null)
          return
        }

        try {
          const updated = JSON.parse(event.newValue) as Funnel
          setFunnel(updated)
        } catch (error) {
          console.error('Unable to refresh preview state from cache', error)
        }
      }
    }

    window.addEventListener('storage', handleStorage)
    return () => window.removeEventListener('storage', handleStorage)
  }, [id])

  useEffect(() => {
    setResponses({})
    setCurrentStepIndex(0)
  }, [funnel?.id])

  useEffect(() => {
    if (!funnel || funnel.steps.length === 0) {
      setCurrentStepIndex(0)
      return
    }

    setCurrentStepIndex((prev) => {
      if (prev >= funnel.steps.length) {
        return funnel.steps.length - 1
      }
      return prev
    })
  }, [funnel])

  const primaryColor = useMemo(() => {
    if (!funnel?.settings?.theme?.primaryColor) {
      return '#0ea5e9'
    }
    return funnel.settings.theme.primaryColor
  }, [funnel?.settings?.theme?.primaryColor])

  const handleNext = () => {
    if (funnel && currentStepIndex < funnel.steps.length - 1) {
      setCurrentStepIndex(currentStepIndex + 1)
    }
  }

  const handleBack = () => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1)
    }
  }

  const handleResponse = (stepId: string, response: any) => {
    setResponses({ ...responses, [stepId]: response })
    handleNext()
  }

  if (isLoading) {
    return <div className="flex h-screen items-center justify-center">Loading preview...</div>
  }

  if (!funnel) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50 px-4 text-center">
        <div>
          <h2 className="mb-2 text-2xl font-semibold text-gray-800">Preview unavailable</h2>
          <p className="text-sm text-gray-500">
            We couldn't find a saved funnel for this preview. Go back to the builder, make sure your
            funnel has at least one step, and click preview again.
          </p>
        </div>
      </div>
    )
  }

  const totalSteps = funnel.steps.length
  const progress = totalSteps === 0 ? 0 : ((currentStepIndex + 1) / totalSteps) * 100
  const currentStep = totalSteps > 0 ? funnel.steps[currentStepIndex] : null

  if (!currentStep) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50 px-4 text-center">
        <div>
          <h2 className="mb-2 text-2xl font-semibold text-gray-800">This funnel is empty</h2>
          <p className="text-sm text-gray-500">
            Add at least one step in the builder, save your changes, and open the preview again to
            see the customer experience.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      {funnel.settings?.progressBar ? (
        <div className="fixed top-0 left-0 right-0 h-1 bg-gray-200">
          <div
            className="h-full bg-primary-600 transition-all duration-300"
            style={{ width: `${progress}%`, backgroundColor: primaryColor }}
          />
        </div>
      ) : null}

      <div className="container mx-auto px-4 py-16">
        <div className="max-w-2xl mx-auto">
          <div className="bg-white rounded-2xl shadow-xl p-8 animate-fade-in">
            <div className="mb-8">
              <h2 className="text-3xl font-bold text-gray-900 mb-2">
                {currentStep.title}
              </h2>
              <p className="text-sm text-gray-500">
                Step {currentStepIndex + 1} of {funnel.steps.length}
              </p>
            </div>

            <div className="mb-8">
              {currentStep.type === 'quiz' && (
                <div>
                  <p className="text-xl mb-6">{(currentStep.content as any).question}</p>
                  <div className="space-y-3">
                    {(currentStep.content as any).options?.map((option: any) => (
                      <button
                        key={option.id}
                        onClick={() => handleResponse(currentStep.id, option.id)}
                        className="w-full p-4 bg-gray-50 hover:bg-primary-50 border-2 border-gray-200 hover:border-primary-400 rounded-lg transition-all text-left flex items-center gap-4"
                      >
                        <span className="text-2xl">{option.icon}</span>
                        <span className="text-lg font-medium">{option.text}</span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {currentStep.type === 'form' && (
                <form onSubmit={(e) => {
                  e.preventDefault()
                  const formData = new FormData(e.currentTarget)
                  const data = Object.fromEntries(formData)
                  handleResponse(currentStep.id, data)
                }}>
                  <div className="space-y-4">
                    {(currentStep.content as any).fields?.map((field: any) => (
                      <div key={field.id}>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          {field.label}
                          {field.required && <span className="text-red-500 ml-1">*</span>}
                        </label>
                        {field.type === 'textarea' ? (
                          <textarea
                            name={field.id}
                            required={field.required}
                            placeholder={field.placeholder}
                            className="input w-full h-24 resize-none"
                          />
                        ) : (
                          <input
                            type={field.type}
                            name={field.id}
                            required={field.required}
                            placeholder={field.placeholder}
                            className="input w-full"
                          />
                        )}
                      </div>
                    ))}
                  </div>
                  <button type="submit" className="btn btn-primary w-full mt-6">
                    {(currentStep.content as any).submitText}
                  </button>
                </form>
              )}

              {currentStep.type === 'content' && (
                <div>
                  <h3 className="text-2xl font-bold mb-4">
                    {(currentStep.content as any).headline}
                  </h3>
                  <p className="text-gray-600 mb-6">
                    {(currentStep.content as any).body}
                  </p>
                  <button
                    onClick={() => handleResponse(currentStep.id, true)}
                    className="btn btn-primary"
                  >
                    {(currentStep.content as any).buttonText}
                  </button>
                </div>
              )}

              {currentStep.type === 'cta' && (
                <div className="text-center">
                  <h3 className="text-3xl font-bold mb-4">
                    {(currentStep.content as any).headline}
                  </h3>
                  {(currentStep.content as any).subheadline && (
                    <p className="text-xl text-gray-600 mb-6">
                      {(currentStep.content as any).subheadline}
                    </p>
                  )}
                  <button
                    onClick={() => window.location.href = (currentStep.content as any).buttonUrl}
                    className="btn btn-primary px-8 py-3 text-lg"
                  >
                    {(currentStep.content as any).buttonText}
                  </button>
                </div>
              )}
            </div>

            <div className="flex justify-between">
              {currentStepIndex > 0 && (
                <button
                  onClick={handleBack}
                  className="btn btn-outline"
                >
                  <ChevronLeft className="w-4 h-4 mr-2" />
                  Back
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default FunnelPreview

import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { Funnel } from '../types'

const FunnelPreview = () => {
  const { id } = useParams()
  const [funnel, setFunnel] = useState<Funnel | null>(null)
  const [currentStepIndex, setCurrentStepIndex] = useState(0)
  const [responses, setResponses] = useState<Record<string, any>>({})

  useEffect(() => {
    // Load funnel data - mock for now
    const mockFunnel: Funnel = {
      id: id!,
      name: 'Sample Funnel',
      slug: 'sample-funnel',
      steps: [
        {
          id: 'step-1',
          type: 'quiz',
          position: 0,
          title: 'What brings you here today?',
          content: {
            question: 'What brings you here today?',
            options: [
              { id: 'opt-1', text: 'Growing my business', icon: '📈' },
              { id: 'opt-2', text: 'Finding new customers', icon: '🎯' },
              { id: 'opt-3', text: 'Improving conversions', icon: '🚀' },
            ]
          }
        },
        {
          id: 'step-2',
          type: 'form',
          position: 1,
          title: 'Let\'s get started!',
          content: {
            fields: [
              { id: 'field-1', type: 'text' as const, label: 'Full Name', required: true },
              { id: 'field-2', type: 'email' as const, label: 'Email', required: true },
              { id: 'field-3', type: 'tel' as const, label: 'Phone', required: false },
            ],
            submitText: 'Get My Free Guide'
          }
        },
        {
          id: 'step-3',
          type: 'cta',
          position: 2,
          title: 'Success!',
          content: {
            headline: 'Check Your Email!',
            subheadline: 'We\'ve sent your free guide to your inbox.',
            buttonText: 'Go to Dashboard',
            buttonUrl: '/dashboard'
          }
        }
      ],
      settings: {
        theme: { primaryColor: '#0ea5e9' },
        progressBar: true
      },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      status: 'active'
    }
    setFunnel(mockFunnel)
  }, [id])

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

  if (!funnel) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>
  }

  const currentStep = funnel.steps[currentStepIndex]
  const progress = ((currentStepIndex + 1) / funnel.steps.length) * 100

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      {funnel.settings.progressBar && (
        <div className="fixed top-0 left-0 right-0 h-1 bg-gray-200">
          <div
            className="h-full bg-primary-600 transition-all duration-300"
            style={{ width: `${progress}%` }}
          />
        </div>
      )}

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
// Windmill App: Scenario Generator V1 - Idea Input Page
// This component provides a simple interface for entering customer requirements
// and generating complete Vrooli scenarios using Claude Code

import { useState } from 'react';
import { 
  Button, 
  TextArea, 
  Select, 
  Card, 
  Alert,
  LoadingSpinner,
  Container,
  Title,
  Text
} from '@windmill/components';

interface GenerationRequest {
  customerInput: string;
  complexity: 'simple' | 'intermediate' | 'advanced';
  category: string;
  additionalContext?: string;
}

interface GenerationResponse {
  scenarioId: string;
  scenarioName: string;
  resourcesRequired: string[];
  estimatedRevenue: {
    min: number;
    max: number;
  };
  generationTimeMs: number;
  status: 'success' | 'error';
  message?: string;
}

export default function ScenarioGeneratorInput() {
  const [customerInput, setCustomerInput] = useState('');
  const [complexity, setComplexity] = useState<'simple' | 'intermediate' | 'advanced'>('intermediate');
  const [category, setCategory] = useState('business-automation');
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationResult, setGenerationResult] = useState<GenerationResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!customerInput.trim()) {
      setError('Please enter customer requirements');
      return;
    }

    setIsGenerating(true);
    setError(null);
    setGenerationResult(null);

    try {
      // Call n8n workflow via webhook
      const response = await fetch('http://localhost:5678/webhook/scenario-generator', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          customerInput,
          complexity,
          category,
          timestamp: new Date().toISOString(),
        } as GenerationRequest),
      });

      if (!response.ok) {
        throw new Error(`Generation failed: ${response.statusText}`);
      }

      const result: GenerationResponse = await response.json();
      setGenerationResult(result);

      // Clear input on success
      if (result.status === 'success') {
        setCustomerInput('');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate scenario');
    } finally {
      setIsGenerating(false);
    }
  };

  const exampleRequirements = [
    "I need a customer support chatbot that can handle returns, track orders, and answer FAQs for my e-commerce store",
    "Build a document processing system that extracts data from invoices and automatically enters it into QuickBooks",
    "Create a social media content generator that creates posts with images for Instagram, Twitter, and LinkedIn based on blog articles",
    "I want an employee onboarding system that sends welcome emails, schedules training, and tracks document completion",
    "Build a real estate listing analyzer that scrapes MLS data, performs comparisons, and generates investment reports",
  ];

  return (
    <Container className="max-w-4xl mx-auto p-6">
      <Title level={1} className="mb-6">
        Scenario Generator V1
      </Title>
      
      <Text className="mb-8 text-gray-600">
        Transform customer requirements into complete, deployable Vrooli scenarios in minutes.
        Each scenario includes all necessary resources, workflows, and UI components.
      </Text>

      <Card className="mb-6">
        <Title level={3} className="mb-4">Customer Requirements</Title>
        
        <TextArea
          value={customerInput}
          onChange={(e) => setCustomerInput(e.target.value)}
          placeholder="Describe what the customer needs. Be specific about features, integrations, and business goals..."
          rows={8}
          className="w-full mb-4"
          disabled={isGenerating}
        />

        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <label className="block text-sm font-medium mb-2">Complexity Level</label>
            <Select
              value={complexity}
              onChange={(e) => setComplexity(e.target.value as any)}
              disabled={isGenerating}
            >
              <option value="simple">Simple (1-3 resources)</option>
              <option value="intermediate">Intermediate (4-6 resources)</option>
              <option value="advanced">Advanced (7+ resources)</option>
            </Select>
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">Category</label>
            <Select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              disabled={isGenerating}
            >
              <option value="business-automation">Business Automation</option>
              <option value="ai-assistant">AI Assistant</option>
              <option value="data-processing">Data Processing</option>
              <option value="customer-service">Customer Service</option>
              <option value="content-generation">Content Generation</option>
              <option value="analytics">Analytics & Reporting</option>
              <option value="e-commerce">E-Commerce</option>
              <option value="education">Education & Training</option>
              <option value="healthcare">Healthcare</option>
              <option value="other">Other</option>
            </Select>
          </div>
        </div>

        <Button
          onClick={handleGenerate}
          disabled={isGenerating || !customerInput.trim()}
          className="w-full"
          variant="primary"
          size="lg"
        >
          {isGenerating ? (
            <>
              <LoadingSpinner className="mr-2" />
              Generating Scenario...
            </>
          ) : (
            'Generate Scenario'
          )}
        </Button>
      </Card>

      {/* Example Requirements */}
      <Card className="mb-6">
        <Title level={3} className="mb-4">Example Requirements</Title>
        <div className="space-y-2">
          {exampleRequirements.map((example, index) => (
            <div
              key={index}
              className="p-3 bg-gray-50 rounded cursor-pointer hover:bg-gray-100 transition-colors"
              onClick={() => setCustomerInput(example)}
            >
              <Text className="text-sm">{example}</Text>
            </div>
          ))}
        </div>
      </Card>

      {/* Error Display */}
      {error && (
        <Alert variant="error" className="mb-6">
          {error}
        </Alert>
      )}

      {/* Success Result */}
      {generationResult && generationResult.status === 'success' && (
        <Alert variant="success" className="mb-6">
          <Title level={4} className="mb-2">Scenario Generated Successfully!</Title>
          <div className="space-y-2">
            <Text><strong>Scenario ID:</strong> {generationResult.scenarioId}</Text>
            <Text><strong>Name:</strong> {generationResult.scenarioName}</Text>
            <Text><strong>Resources:</strong> {generationResult.resourcesRequired.join(', ')}</Text>
            <Text>
              <strong>Revenue Potential:</strong> ${generationResult.estimatedRevenue.min.toLocaleString()} - 
              ${generationResult.estimatedRevenue.max.toLocaleString()} per deployment
            </Text>
            <Text><strong>Generation Time:</strong> {(generationResult.generationTimeMs / 1000).toFixed(2)} seconds</Text>
          </div>
          <Button
            onClick={() => window.location.href = '/scenarios'}
            className="mt-4"
            variant="secondary"
          >
            View Generated Scenarios
          </Button>
        </Alert>
      )}
    </Container>
  );
}
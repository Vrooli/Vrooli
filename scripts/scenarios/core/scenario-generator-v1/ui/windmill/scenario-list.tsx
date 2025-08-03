// Windmill App: Scenario Generator V1 - Scenario List Page
// This component displays all generated scenarios with management capabilities

import { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Card,
  Badge,
  Container,
  Title,
  Text,
  SearchInput,
  Select,
  Alert,
  Modal,
  LoadingSpinner
} from '@windmill/components';

interface Scenario {
  id: string;
  name: string;
  customerInput: string;
  generatedAt: string;
  complexity: 'simple' | 'intermediate' | 'advanced';
  category: string;
  resourcesRequired: string[];
  estimatedRevenue: {
    min: number;
    max: number;
  };
  status: 'ready' | 'deployed' | 'draft' | 'failed';
  generationTimeMs: number;
  downloadUrl?: string;
}

export default function ScenarioList() {
  const [scenarios, setScenarios] = useState<Scenario[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [selectedScenario, setSelectedScenario] = useState<Scenario | null>(null);
  const [showDetailsModal, setShowDetailsModal] = useState(false);
  const [deploymentStatus, setDeploymentStatus] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    fetchScenarios();
    // Poll for updates every 30 seconds
    const interval = setInterval(fetchScenarios, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchScenarios = async () => {
    try {
      // Fetch from PostgreSQL via n8n webhook
      const response = await fetch('http://localhost:5678/webhook/get-scenarios', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch scenarios');
      }

      const data = await response.json();
      setScenarios(data.scenarios || []);
    } catch (error) {
      console.error('Error fetching scenarios:', error);
      // Use mock data for demonstration
      setScenarios(getMockScenarios());
    } finally {
      setLoading(false);
    }
  };

  const getMockScenarios = (): Scenario[] => [
    {
      id: 'scn-001',
      name: 'E-Commerce Customer Support Bot',
      customerInput: 'Customer support chatbot for handling returns and FAQs',
      generatedAt: new Date(Date.now() - 86400000).toISOString(),
      complexity: 'intermediate',
      category: 'customer-service',
      resourcesRequired: ['ollama', 'n8n', 'postgres', 'redis'],
      estimatedRevenue: { min: 15000, max: 25000 },
      status: 'ready',
      generationTimeMs: 45000,
    },
    {
      id: 'scn-002',
      name: 'Invoice Processing Pipeline',
      customerInput: 'Extract data from invoices and sync with QuickBooks',
      generatedAt: new Date(Date.now() - 172800000).toISOString(),
      complexity: 'advanced',
      category: 'data-processing',
      resourcesRequired: ['unstructured-io', 'n8n', 'postgres', 'minio', 'windmill'],
      estimatedRevenue: { min: 20000, max: 35000 },
      status: 'deployed',
      generationTimeMs: 62000,
    },
  ];

  const handleDeploy = async (scenarioId: string) => {
    setDeploymentStatus({ ...deploymentStatus, [scenarioId]: 'deploying' });
    
    try {
      const response = await fetch('http://localhost:5678/webhook/deploy-scenario', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ scenarioId }),
      });

      if (!response.ok) {
        throw new Error('Deployment failed');
      }

      setDeploymentStatus({ ...deploymentStatus, [scenarioId]: 'deployed' });
      // Refresh scenarios
      await fetchScenarios();
    } catch (error) {
      setDeploymentStatus({ ...deploymentStatus, [scenarioId]: 'failed' });
      console.error('Deployment error:', error);
    }
  };

  const handleDownload = async (scenarioId: string) => {
    try {
      const response = await fetch(`http://localhost:9000/generated-scenarios/${scenarioId}.zip`);
      if (!response.ok) throw new Error('Download failed');
      
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${scenarioId}.zip`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Download error:', error);
    }
  };

  const handleValidate = async (scenarioId: string) => {
    try {
      const response = await fetch('http://localhost:5678/webhook/validate-scenario', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ scenarioId }),
      });

      const result = await response.json();
      alert(`Validation ${result.valid ? 'passed' : 'failed'}: ${result.message}`);
    } catch (error) {
      console.error('Validation error:', error);
    }
  };

  // Filter scenarios
  const filteredScenarios = scenarios.filter(scenario => {
    const matchesSearch = scenario.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         scenario.customerInput.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = categoryFilter === 'all' || scenario.category === categoryFilter;
    const matchesStatus = statusFilter === 'all' || scenario.status === statusFilter;
    return matchesSearch && matchesCategory && matchesStatus;
  });

  const getStatusBadge = (status: string) => {
    const variants: { [key: string]: 'success' | 'warning' | 'error' | 'info' } = {
      ready: 'success',
      deployed: 'info',
      draft: 'warning',
      failed: 'error',
    };
    return <Badge variant={variants[status] || 'info'}>{status}</Badge>;
  };

  const getComplexityBadge = (complexity: string) => {
    const colors: { [key: string]: 'green' | 'yellow' | 'red' } = {
      simple: 'green',
      intermediate: 'yellow',
      advanced: 'red',
    };
    return <Badge color={colors[complexity] || 'gray'}>{complexity}</Badge>;
  };

  return (
    <Container className="max-w-7xl mx-auto p-6">
      <div className="flex justify-between items-center mb-6">
        <Title level={1}>Generated Scenarios</Title>
        <Button
          onClick={() => window.location.href = '/generate'}
          variant="primary"
        >
          Generate New Scenario
        </Button>
      </div>

      {/* Filters */}
      <Card className="mb-6">
        <div className="grid grid-cols-3 gap-4">
          <SearchInput
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search scenarios..."
          />
          <Select
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
          >
            <option value="all">All Categories</option>
            <option value="business-automation">Business Automation</option>
            <option value="customer-service">Customer Service</option>
            <option value="data-processing">Data Processing</option>
            <option value="content-generation">Content Generation</option>
            <option value="analytics">Analytics</option>
          </Select>
          <Select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="all">All Statuses</option>
            <option value="ready">Ready</option>
            <option value="deployed">Deployed</option>
            <option value="draft">Draft</option>
            <option value="failed">Failed</option>
          </Select>
        </div>
      </Card>

      {/* Scenarios Table */}
      {loading ? (
        <div className="flex justify-center py-12">
          <LoadingSpinner size="lg" />
        </div>
      ) : filteredScenarios.length === 0 ? (
        <Alert variant="info">
          No scenarios found. Generate your first scenario to get started!
        </Alert>
      ) : (
        <Card>
          <Table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Category</th>
                <th>Complexity</th>
                <th>Resources</th>
                <th>Revenue Potential</th>
                <th>Status</th>
                <th>Generated</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredScenarios.map((scenario) => (
                <tr key={scenario.id}>
                  <td>
                    <button
                      onClick={() => {
                        setSelectedScenario(scenario);
                        setShowDetailsModal(true);
                      }}
                      className="text-blue-600 hover:underline font-medium"
                    >
                      {scenario.name}
                    </button>
                  </td>
                  <td>{scenario.category.replace('-', ' ')}</td>
                  <td>{getComplexityBadge(scenario.complexity)}</td>
                  <td>
                    <div className="flex flex-wrap gap-1">
                      {scenario.resourcesRequired.slice(0, 3).map(resource => (
                        <Badge key={resource} variant="secondary" size="sm">
                          {resource}
                        </Badge>
                      ))}
                      {scenario.resourcesRequired.length > 3 && (
                        <Badge variant="secondary" size="sm">
                          +{scenario.resourcesRequired.length - 3}
                        </Badge>
                      )}
                    </div>
                  </td>
                  <td>
                    ${scenario.estimatedRevenue.min.toLocaleString()} - 
                    ${scenario.estimatedRevenue.max.toLocaleString()}
                  </td>
                  <td>{getStatusBadge(scenario.status)}</td>
                  <td>{new Date(scenario.generatedAt).toLocaleDateString()}</td>
                  <td>
                    <div className="flex gap-2">
                      {scenario.status === 'ready' && (
                        <Button
                          size="sm"
                          variant="primary"
                          onClick={() => handleDeploy(scenario.id)}
                          disabled={deploymentStatus[scenario.id] === 'deploying'}
                        >
                          {deploymentStatus[scenario.id] === 'deploying' ? (
                            <LoadingSpinner size="sm" />
                          ) : (
                            'Deploy'
                          )}
                        </Button>
                      )}
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => handleDownload(scenario.id)}
                      >
                        Download
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleValidate(scenario.id)}
                      >
                        Validate
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Card>
      )}

      {/* Details Modal */}
      {showDetailsModal && selectedScenario && (
        <Modal
          isOpen={showDetailsModal}
          onClose={() => setShowDetailsModal(false)}
          title={selectedScenario.name}
        >
          <div className="space-y-4">
            <div>
              <Text className="font-semibold">Customer Requirements:</Text>
              <Text className="mt-1">{selectedScenario.customerInput}</Text>
            </div>
            <div>
              <Text className="font-semibold">Resources Required:</Text>
              <div className="flex flex-wrap gap-2 mt-1">
                {selectedScenario.resourcesRequired.map(resource => (
                  <Badge key={resource} variant="secondary">
                    {resource}
                  </Badge>
                ))}
              </div>
            </div>
            <div>
              <Text className="font-semibold">Generation Metrics:</Text>
              <Text className="mt-1">
                Time: {(selectedScenario.generationTimeMs / 1000).toFixed(2)} seconds
              </Text>
            </div>
            <div className="flex gap-2 pt-4">
              <Button
                variant="primary"
                onClick={() => handleDeploy(selectedScenario.id)}
              >
                Deploy Scenario
              </Button>
              <Button
                variant="secondary"
                onClick={() => handleDownload(selectedScenario.id)}
              >
                Download Files
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </Container>
  );
}
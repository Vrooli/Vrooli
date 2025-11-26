====================
Vrooli Resource SDK
====================

.. image:: https://img.shields.io/badge/version-2.1.0-blue.svg
   :target: https://github.com/Vrooli/Vrooli/releases
   :alt: Version

.. image:: https://img.shields.io/badge/python-3.8+-blue.svg
   :target: https://www.python.org/downloads/
   :alt: Python Version

.. image:: https://img.shields.io/badge/license-MIT-green.svg
   :target: https://opensource.org/licenses/MIT
   :alt: License

A comprehensive Python SDK for managing and orchestrating AI resources, automation workflows, and system integrations in the Vrooli ecosystem.

Overview
========

The Vrooli Resource SDK provides a unified interface for interacting with various AI and automation services, including:

* **AI Services**: Local LLMs (Ollama), speech recognition (Whisper), document processing
* **Automation Platforms**: Node-RED flows, Huginn agents
* **Agent Services**: Browser automation (Browserless), screen interaction (Agent-S2)
* **Storage Solutions**: MinIO object storage, Redis caching, PostgreSQL databases
* **Search Services**: SearXNG privacy-respecting search aggregation

Features
========

🚀 **Dynamic Resource Discovery**
   Automatically detect and configure available services in your environment

🔍 **Health Monitoring**
   Real-time health checks and performance metrics for all resources

⚡ **Task Execution**
   Execute tasks across different resource types with unified error handling

🔄 **Workflow Orchestration**
   Chain multiple resources together in complex automation workflows  

📊 **Comprehensive Logging**
   Structured logging with configurable levels and output formats

🛡️ **Error Resilience**
   Automatic retries, circuit breakers, and graceful degradation

Installation
============

Install from PyPI:

.. code-block:: bash

   pip install vrooli-resource-sdk

Install from source:

.. code-block:: bash

   git clone https://github.com/Vrooli/Vrooli.git
   cd Vrooli/scripts/resources/sdk/python
   pip install -e .

Requirements
------------

* Python 3.8+
* Redis (for caching and coordination)
* One or more Vrooli-compatible resources

Quick Start
===========

Basic Usage
-----------

.. code-block:: python

   from vrooli_sdk import ResourceManager
   
   # Initialize the resource manager
   manager = ResourceManager()
   
   # Discover available resources
   resources = await manager.discover_resources()
   print(f"Found {len(resources)} resources")
   
   # Get a specific resource
   ollama = manager.get_resource('ollama')
   if ollama and ollama.is_healthy():
       # Execute a task
       result = await ollama.execute_task('generate_text', {
           'prompt': 'Explain quantum computing',
           'model': 'llama3.1:8b',
           'max_tokens': 500
       })
       print(result.data['text'])

Configuration
-------------

.. code-block:: python

   from vrooli_sdk import ResourceManager, ResourceConfig
   
   config = ResourceConfig(
       discovery_enabled=True,
       health_check_interval=60,
       retry_attempts=3,
       timeout=30000,
       log_level='INFO'
   )
   
   manager = ResourceManager(config=config)

Advanced Usage
==============

Custom Resource Types
---------------------

Create custom resource implementations:

.. code-block:: python

   from vrooli_sdk import BaseResource, ResourceCapability
   
   class CustomAIResource(BaseResource):
       resource_type = 'ai'
       
       def __init__(self, config):
           super().__init__(config)
           self.capabilities = [
               ResourceCapability('text_analysis', self.analyze_text),
               ResourceCapability('summarization', self.summarize)
           ]
       
       async def analyze_text(self, text: str, **kwargs):
           # Custom implementation
           return {'sentiment': 'positive', 'entities': [...]}
       
       async def summarize(self, text: str, max_length: int = 100):
           # Custom implementation
           return {'summary': '...'}

Workflow Orchestration
----------------------

Chain multiple resources in workflows:

.. code-block:: python

   from vrooli_sdk import WorkflowBuilder
   
   # Build a document processing workflow
   workflow = (WorkflowBuilder()
       .add_step('extract', 'unstructured-io', 'process_document')
       .add_step('analyze', 'ollama', 'generate_text', {
           'prompt': 'Analyze this document: {{extract.text}}'
       })
       .add_step('store', 'minio', 'upload_object', {
           'bucket': 'analyses',
           'content': '{{analyze.result}}'
       })
   )
   
   # Execute the workflow
   result = await manager.execute_workflow(workflow, {
       'document_path': '/path/to/document.pdf'
   })

Error Handling
--------------

.. code-block:: python

   from vrooli_sdk.exceptions import (
       ResourceNotFoundError,
       ResourceUnavailableError,
       TaskExecutionError
   )
   
   try:
       result = await resource.execute_task('complex_task', params)
   except ResourceNotFoundError:
       print("Resource not configured")
   except ResourceUnavailableError:
       print("Resource is currently unavailable")
   except TaskExecutionError as e:
       print(f"Task failed: {e.message}")
       print(f"Error details: {e.details}")

Real-time Monitoring
--------------------

Monitor resource health and performance:

.. code-block:: python

   from vrooli_sdk import ResourceMonitor
   
   monitor = ResourceMonitor(manager)
   
   @monitor.on_health_change
   async def handle_health_change(resource_id, status):
       if not status.healthy:
           print(f"Resource {resource_id} is unhealthy: {status.error}")
           await manager.restart_resource(resource_id)
   
   @monitor.on_performance_alert
   async def handle_performance_alert(resource_id, metrics):
       if metrics.response_time > 5000:
           print(f"Slow response from {resource_id}: {metrics.response_time}ms")
   
   # Start monitoring
   await monitor.start()

API Reference
=============

ResourceManager
---------------

.. autoclass:: vrooli_sdk.ResourceManager
   :members:
   :undoc-members:
   :show-inheritance:

BaseResource
------------

.. autoclass:: vrooli_sdk.BaseResource
   :members:
   :undoc-members:
   :show-inheritance:

TaskResult
----------

.. autoclass:: vrooli_sdk.TaskResult
   :members:
   :undoc-members:
   :show-inheritance:

Configuration
=============

Environment Variables
--------------------

.. list-table::
   :header-rows: 1
   :widths: 30 20 50

   * - Variable
     - Default
     - Description
   * - ``VROOLI_CONFIG_PATH``
     - ``~/.vrooli/config.json``
     - Path to configuration file
   * - ``VROOLI_LOG_LEVEL``
     - ``INFO``
     - Logging level (DEBUG, INFO, WARN, ERROR)
   * - ``VROOLI_DISCOVERY_ENABLED``
     - ``true``
     - Enable automatic resource discovery
   * - ``VROOLI_HEALTH_CHECK_INTERVAL``
     - ``60``
     - Health check interval in seconds
   * - ``VROOLI_RETRY_ATTEMPTS``
     - ``3``
     - Number of retry attempts for failed requests
   * - ``VROOLI_TIMEOUT``
     - ``30000``
     - Default timeout in milliseconds

Configuration File
------------------

Create a configuration file at ``~/.vrooli/config.json``:

.. code-block:: json

   {
     "discovery": {
       "enabled": true,
       "interval": 300,
       "timeout": 10000
     },
     "health_checks": {
       "enabled": true,
       "interval": 60,
       "timeout": 5000,
       "retries": 3
     },
     "logging": {
       "level": "INFO",
       "format": "structured",
       "output": "console"
     },
     "resources": {
       "ollama": {
         "enabled": true,
         "base_url": "http://localhost:11434",
         "timeout": 30000
       }
     }
   }

Resource Types
==============

AI Resources
------------

Ollama
~~~~~~

Local LLM inference with multiple model support.

**Capabilities:**
- ``generate_text``: Generate text from prompts
- ``analyze_image``: Analyze images with vision models
- ``embed_text``: Generate text embeddings

**Configuration:**

.. code-block:: python

   {
     "base_url": "http://localhost:11434",
     "models": ["llama3.1:8b", "qwen2.5-coder:7b"],
     "timeout": 30000,
     "max_concurrent": 4
   }

Whisper
~~~~~~~

Speech-to-text transcription service.

**Capabilities:**
- ``transcribe_audio``: Convert audio to text
- ``detect_language``: Identify audio language

**Configuration:**

.. code-block:: python

   {
     "base_url": "http://localhost:8090",
     "model_size": "base",
     "supported_formats": ["wav", "mp3", "ogg", "m4a"]
   }

Automation Resources
--------------------

Node-RED
~~~~~~~~

Real-time flow programming for IoT and automation.

**Capabilities:**
- ``trigger_flow``: Trigger Node-RED flow
- ``get_flow_data``: Retrieve flow execution data

Testing
=======

Run the test suite:

.. code-block:: bash

   # Install test dependencies
   pip install -e ".[test]"
   
   # Run tests
   pytest
   
   # Run with coverage
   pytest --cov=vrooli_sdk --cov-report=html
   
   # Run integration tests (requires running resources)
   pytest tests/integration/

Test with Docker:

.. code-block:: bash

   # Start test environment
   docker-compose -f docker-compose.test.yml up -d
   
   # Run tests
   docker-compose -f docker-compose.test.yml exec test pytest
   
   # Clean up
   docker-compose -f docker-compose.test.yml down

Examples
========

The ``examples/`` directory contains comprehensive usage examples:

- ``basic_usage.py``: Simple resource discovery and task execution
- ``workflow_orchestration.py``: Complex multi-step workflows
- ``health_monitoring.py``: Real-time resource monitoring
- ``custom_resources.py``: Creating custom resource types
- ``error_handling.py``: Comprehensive error handling strategies

Contributing
============

We welcome contributions! Please see our `Contributing Guide`_ for details.

Development Setup
-----------------

.. code-block:: bash

   # Clone the repository
   git clone https://github.com/Vrooli/Vrooli.git
   cd Vrooli/scripts/resources/sdk/python
   
   # Create virtual environment
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\\Scripts\\activate
   
   # Install in development mode
   pip install -e ".[dev]"
   
   # Install pre-commit hooks
   pre-commit install
   
   # Run tests
   pytest

Code Style
-----------

We use:

- `black`_ for code formatting
- `isort`_ for import sorting  
- `flake8`_ for linting
- `mypy`_ for type checking

Run all checks:

.. code-block:: bash

   # Format code
   black src tests
   isort src tests
   
   # Check linting
   flake8 src tests
   
   # Type checking
   mypy src

License
=======

This project is licensed under the MIT License - see the `LICENSE`_ file for details.

Changelog
=========

See `CHANGELOG.md`_ for a detailed history of changes.

Support
=======

- 📖 **Documentation**: https://docs.vrooli.com
- 🐛 **Issues**: https://github.com/Vrooli/Vrooli/issues
- 💬 **Discussions**: https://github.com/Vrooli/Vrooli/discussions
- 📧 **Email**: support@vrooli.com

Acknowledgments
===============

Special thanks to the open-source projects that make Vrooli possible:

- `Ollama`_ for local LLM inference
- `Node-RED`_ for workflow automation
- `FastAPI`_ for the web framework
- `Pydantic`_ for data validation

.. _Contributing Guide: https://github.com/Vrooli/Vrooli/blob/main/CONTRIBUTING.md
.. _LICENSE: https://github.com/Vrooli/Vrooli/blob/main/LICENSE
.. _CHANGELOG.md: https://github.com/Vrooli/Vrooli/blob/main/CHANGELOG.md
.. _black: https://black.readthedocs.io/
.. _isort: https://pycqa.github.io/isort/
.. _flake8: https://flake8.pycqa.org/
.. _mypy: https://mypy.readthedocs.io/
.. _Ollama: https://ollama.ai/
.. _Node-RED: https://nodered.org/
.. _FastAPI: https://fastapi.tiangolo.com/
.. _Pydantic: https://pydantic-docs.helpmanual.io/

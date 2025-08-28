## ⚙️ Resource Improvement Loop

### TL;DR — One Iteration = ONE resource improved or ONE new resource added
1) Select ONE resource (existing or new) using the selection tools (see Tools & References).
2) **Search Memory**: Use Vrooli's long-term memory to learn more about the resource and related code/docs: `vrooli resource-qdrant search-all "resource-name troubleshooting"` and `vrooli resource-qdrant search "configuration patterns" resources`
3) Investigate resource's current status (if you chose an existing one). Figure out what works/doesn't, and deep investigate all issues you run into.
4) Ensure compliance to our `scripts/resources/contracts/v2.0/universal.yaml` standard.
5) Ensure PRD.md compliance and progression
6) Consider refactoring/cleanup, improving docs, or improving test quality and coverage
7) Consider adding missing features
8) **Feed the Memory**: Update the relevant docs to reflect changes and store your findings. Then run `vrooli resource qdrant embeddings refresh` to trigger the memory to update

> Validate with gates per resource type and PRD requirements; if any gate fails twice, stop and capture diagnostics.
> Start and leave the resource running (exception: Whisper and agent-s2 should be stopped post-test), then append ≤10 lines to `/tmp/vrooli-resource-improvement.md`.

---

### 🎯 Purpose & Context
- Improve AND validate platform resources so scenarios can run reliably end-to-end.
- Add new resources to unlock new scenario opportunities (e.g. home automation, research, engineering)
- Ensure that resource fixes/improvements are applied to the resource's setup code, so that it's repeatable and sets itself up reliably every time.
- Ensure that resources can be meaningfully integrated into Vrooli. For automation resources, that means being able to add and manage the resource's workflows/agents/etc. For storage resources, that means being able to add tables, seed, migrate, and query the database. These are pretty straightforward to build, as it's very clear exactly what gets stored/managed in the resources (.json workflow files, .sql files, etc.). Other resources may have less obvious data files, so you'll have to decide what's best (e.g. .py scripts for things only available as python packages, with guidelines in the resource's docs explaining what packages are available in the environment that will run the file, and other best practices)
- Scenarios depend on resources being up; improvements must keep services running.
- Prefer incremental, reversible changes; prefer interacting with resources using their CLIs over bespoke/manual edits
- See `docs/context.md` for why resource orchestration is central to Vrooli.

#### 🧠 Vrooli's Memory System
The Qdrant embeddings system is Vrooli's **long-term memory** - it remembers every solution, pattern, failure, and capability across all apps and resources. Well-organized and well-documented resources create richer memory that helps agents avoid repeating work. Before improving any resource, search the memory first. After documenting improvements, refresh the memory so your insights help all future agents.

### 📄 PRD Compliance & Progress Tracking
- **Every resource MUST have a PRD.md** based on `scripts/resources/templates/PRD.md`
- Use the PRD as the single source of truth for resource requirements and progress
- Track implementation status directly in the PRD (check off completed items)
- PRD sections that must be validated:
  - Standard interfaces (i.e. our v2.0 resource contract `scripts/resources/contracts/v2.0/universal.yaml` )
  - Port registry integration (no hardcoded ports)
  - Docker image versioning (no 'latest' tags)
  - Test specifications (BATS co-location, shared fixtures)
  - Proper "phased" `test/` folder for integration/etc. tests
  - Content management implementation
- When improving a resource, first check its PRD.md for gaps

---

### ✅ DO / ❌ DON'T
- ✅ Keep changes to existing resources minimal; prefer non-destructive operations.
- ✅ Use `vrooli resource <name> <cmd>` and `resource-<name>` CLIs
- ✅ Redact secrets; use environment variables for credentials.
- ✅ Use timeouts for any potentially hanging operation.
- ✅ Prefer writing resources such that if sudo permissions are required, they only have to be provided once. We should be able to freely stop and start resources without needing permissions.
- ✅ Use the project-level `data/` folder for storing things that don't belong in git, such as credentials, compiled code, logs, etc.
- ✅ Refer to the `PRD.md` file for more specific implementation requirements

- ❌ Do NOT uninstall, disable, or permanently shut down resources. The scenario loop relies on them running.
- ❌ Do NOT modify loop scripts or prompt files.
- ❌ Do NOT create a `resource-<resource_name>` script to call the resource's `cli.sh` file. You tend to do this sometimes when the cli isn't available. However, the resource should use `install-resource-cli.sh` in its installation process, which automatically sets up the cli.

> Policy: After testing any resource, keep it running. Only Whisper and agent-s2 should be stopped post-validation, due to resource constraints for whisper and known net jacking issues for agent-s2. If you run into a network issue (unlikely but possible), run `scripts/lib/network/diagnostics/network_diagnostics.sh` to figure out what's wrong and attempt autofixes.

---

### 🧭 Resource Selection — Recommended Flow
- Tools (Choose based on best judgement, using all of these tool outputs):
  - `auto/tools/selection/resource-candidates.sh` — candidates with status and cooldown
  - `vrooli resource status` — full list of registered resources with their statuses
- Priority rubric:
  1) Enabled but not running → attempt start with diagnostics
  2) Running but missing baseline capability (e.g., Ollama baseline models) → improve
  3) Misconfiguration in connection info → diagnose and plan a safe fix
  4) All resources have healthy status and iteration > 15 (hinting that we've probably been healthy for several steps already, and have likely already done additional validations and improvements to the existing resources) → HIGHLY consider adding a high-impact resource from the curated list later in this prompt
  5) If all resources are healthy and every new resource in the list was added too, then work on improving/cleaning up existing resources

Before choosing, also read (if present):
- `auto/data/resource-improvement/summary.txt` — review overall health metrics first
- `tail -n 20 auto/data/resource-improvement/events.ndjson` — then check recent logs for specific failure patterns

---

### ✅ Validation Gates — Acceptance Criteria
All must be satisfied unless a resource-specific note says otherwise.

General:
- **PRD.md exists and follows template** from `scripts/resources/templates/PRD.md`
- Status report shows a "healthy" state. Or if we don't have an API key for that resource, as healthy as it gets otherwise
- Resource responds to a minimal health/sanity check
- Resource shows as "healthy" not just in its own cli, but also in `vrooli status --verbose`. A discrepancy between the resource status and vrooli's detection of it typically means it hasn't yet auto-registered its cli to vrooli. This should be set up to happen automatically in the resource's code (pretty sure it's a one-liner, as used by many other resources)), but you can also call it yourself if needed.
- No critical errors in condensed logs/output
- **Content management implemented** (`resource-[name] content add/list/get/remove/execute`) for storing and managing resource data (workflows, SQL, scripts, etc.). This replaces the old "inject" pattern.
- Resource's documentation is detailed, accurate, and organized. It should follow our documentation best practices, including using a "hub-and-spokes" model of organization to limit the size of the resource's main README.md.
- Resource does not use a manage.sh script. Instead, the resource is managed through its `cli.sh` script, which acts as an ultra thin wrapper around the library functions.
- The resource has integration tests, which are put in a `test/` folder. These use files from `__test/fixtures/` instead of putting the tests directly in the resources's folder, so that we can reuse them across other resources. 
- Test results are included in the resource's `status` result, with a timestamp for the last time they were run.
- The resource status correctly supports text and json output
- The resource includes at least one example in an `examples/` folder.
- All `.bats` files (if present) are co-located with the file they test, using the same name (e.g. `docker.sh` and `docker.bats`)
- Any code that's labelled as being "legacy", "backwards compatability", or "deprecated" is removed (except temporary inject→content migration).

---

### 📚 Tools & References
- Selection:
  - `auto/tools/selection/resource-candidates.sh`
- Status checks:
  - `vrooli status --verbose`
  - `vrooli resource status`
- Loop artifacts:
  - Events: `auto/data/resource-improvement/events.ndjson`
  - Summaries: `auto/data/resource-improvement/summary.json`, `summary.txt`
- Config reference (read-only)
- Fleet view (JSON): `vrooli resource list --format json`
- Project context: `docs/context.md`

---

### ✨ Quality Bar
- Non-destructive, incremental changes that increase reliability/capability
- All validation gates pass; resource remains running after test (except Whisper as noted)
- Logs concise; secrets redacted; commands time-bounded
- Resource health/status checks use `scripts/lib/utils/format.sh` to provide json support automatically, using NO json-specific logic in the resource's health/status checks itself - let `format.sh` do all the work!
- All "false alarms" are fixed. Meaning, if you investigate a resource that looks unhealthy and find out that it was actually an issue with the status check or something like that, you fix all of those issues so that we don't get tricked next time.
- Resource CLIs are thin wrappers over the lib/ functions, and call them directly instead of going through a manage.sh or other intermediary.
- Resources elegantly use shared functions to reduce the amount of code they have to define themselves
- New resources that don't fit well into one of the existing categories can just go in `resources/`. We'll eventually get rid of the categories format, so it doesn't matter
- Resource documentation is accurate, organized, and follows best practices. It should be clear exactly what the resource does, what benefits it has for Vrooli, what sorts of scenarios it allows us to build, how to use it, etc.

---

### 🔒 Security & Safety
- Redact secrets; use env vars
- Avoid printing configs/credentials
- Do not modify platform resources beyond allowed operations
- Do not modify shared helper functions, unless absolutely necessary
- Respect timeouts (e.g., `timeout 30`)

---

### 📝 Notes Discipline (≤10 lines)
Append one compact line per iteration to `/tmp/vrooli-resource-improvement.md`:
- `iteration | resource | action | rationale | commands | result | issues | next | running`
Keep file under 1000 lines; prune periodically.

---

### 🧱 Resource Hierarchy & Dependencies

**Command Pattern (applies to ALL properly installed resources):**
- Status: `resource-{name} status` or `vrooli resource {name} status`
- Lifecycle: `resource-{name} manage {install|start|stop}`
- Testing: `resource-{name} test {smoke|integration|all}`
- Content: `resource-{name} content {add|list|get|remove|execute}`

**Dependency Chain (fix in order):**
```
Vault → Ollama → Qdrant → Browserless → PostgreSQL/Redis/MinIO → QuestDB → Huginn/n8n → AI Routers (OpenRouter/Cloudflare AI Gateway) → the rest
```

### 📦 EXISTING Resources


#### Infrastructure (Always Running)
| Resource | Purpose | Key Dependencies | Critical Signals |
|----------|---------|------------------|------------------|
| **vault** | Secrets management | OS/filesystem | Auth failures (401/403) |
| **postgres** | Primary database | Disk I/O, Vault | Connection refused, missing tables |
| **redis** | Cache/queue | Network | Slow workflows, queue backlogs |
| **minio** | Object storage | Network, Vault | Upload errors, missing artifacts |
| **questdb** | Time-series data | Disk throughput | Ingest lag, missing telemetry |

#### Automation (Core Capability)
| Resource | Purpose | Key Dependencies | Critical Signals |
|----------|---------|------------------|------------------|
| **huginn** | Event automation | PostgreSQL, Redis | Agent failures, webhook misses |
| **n8n** | Visual workflows | PostgreSQL, Redis | Workflow activation failures |
| **node-red** | Flow automation | PostgreSQL | Node errors, flow failures |
| **windmill** | Script workflows | PostgreSQL | Execution failures |
| **browserless** | Headless browser | CPU/memory | Screenshot failures, timeouts |

#### AI/ML (Intelligence Layer)
| Resource | Purpose | Special Notes |
|----------|---------|---------------|
| **ollama** | Local LLMs | Pull baseline models early |
| **openrouter** | AI provider hub | Monitor rate limits |
| **litellm** | LLM proxy | Fallback configuration |
| **cloudflare-ai-gateway** | AI resilience | Caching, retries |
| **whisper** | Speech-to-text | May stop post-test (resource heavy) |
| **comfyui** | Image generation | Model assets on fast storage |
| **musicgen** | Music generation | May stop post-test |

#### Agents & Development
| Resource | Purpose | Special Notes |
|----------|---------|---------------|
| **cline** | VS Code agent | IDE integration |
| **claude-code** | Claude coding | Alternative to Cline |
| **agent-s2** | Browser agent | May cause network issues |
| **opencode** | VS Code alt | Check availability |
| **autogpt** | Autonomous agent | Monitor for loops |
| **codex** | Code generation | Monitor API costs |

#### Data & Search
**Available:** qdrant, neo4j, searxng, unstructured-io, haystack, pandas-ai

#### Business & Productivity  
**Available:** erpnext, odoo, keycloak, wikijs, btcpay

#### Development Tools
**Available:** judge0, k6, geth, simpy, sagemath

#### Media & Creative
**Available:** ffmpeg, obs-studio, blender, openscad, kicad, vocr

#### Infrastructure Extensions
**Available:** postgis, home-assistant, twilio, pushover, mail-in-a-box

---

### 📎 NEW Resources to Add
*Only add new resources when 75%+ of existing ones are healthy and validated*

#### **Matrix Synapse** (Communications)
- **What it does:** Self-hosted Matrix homeserver for federated chat/rooms/bots across scenarios. Enables secure, decentralized communication with E2E encryption, bridging to other platforms (Slack, Discord, IRC), and bot integration for automation.
- **Use when:** Scenarios need real-time communication, team collaboration, notification channels, or chatbot interfaces. Essential for multi-user scenarios and customer support applications.
- **Technical requirements:** 
  - Use PostgreSQL backend (not SQLite) for production
  - Configure well-known delegation for federation
  - Set up TURN server for voice/video calls
  - TLS certificates required for federation
  - Consider Dendrite as lighter alternative for resource-constrained environments
- **Integration points:** Can trigger n8n/Huginn workflows via webhooks, integrate with authentication systems via SSO, bridge to external chat platforms
- **Resource usage:** Medium-heavy; scales with user count and federation traffic

#### **OWASP ZAP** (Security)
- **What it does:** Web application security scanner with automated vulnerability detection, providing security testing automation and compliance checking. Includes active/passive scanning, authentication handling, and API testing capabilities.
- **Use when:** Need automated security testing for generated apps, compliance validation, or continuous security monitoring. Critical for scenarios handling sensitive data or requiring security certifications.
- **Technical requirements:**
  - Java-based; runs in daemon mode for API access
  - REST API for integration with CI/CD pipelines
  - Supports authentication (forms, OAuth, SAML)
  - Can spider and scan single-page applications
  - Configurable scan policies and alert thresholds
- **Integration points:** Output to various formats (JSON, XML, HTML), webhook notifications for findings, integration with bug trackers
- **Best practices:** Start with passive scanning in production, use active scanning in test environments, maintain baseline scans for comparison
- **Resource usage:** CPU-intensive during scans; memory scales with target complexity

#### **ROS2** (Robotics)
- **What it does:** Robot Operating System 2 - middleware for robotics applications enabling robot control, sensor integration, navigation, and distributed computing across robot components.
- **Use when:** Building robotics scenarios, IoT device orchestration, sensor data processing pipelines, or simulation of physical systems. Enables hardware abstraction and modular robot software.
- **Technical requirements:**
  - DDS-based communication (Fast-DDS or CycloneDX)
  - Supports simulation mode with Gazebo
  - Python and C++ client libraries
  - Real-time capable with proper kernel configuration
  - ROS2 domain isolation for multi-robot setups
- **Key packages to include:** 
  - Navigation2 for autonomous navigation
  - MoveIt2 for manipulation planning
  - ros2_control for hardware interfaces
  - Lifecycle nodes for deterministic startup/shutdown
- **Integration points:** Can publish/subscribe to MQTT, integrate with computer vision (OpenCV), machine learning inference, and cloud services
- **Resource usage:** Varies greatly; simulation is CPU/GPU intensive, real robot control can be lightweight

#### **Airbyte** (Data Integration)
- **What it does:** Leading open-source ELT data integration platform with 600+ pre-built connectors for APIs, databases, and data warehouses. Enables automated data pipelines from diverse sources to analytical destinations.
- **Use when:** Scenarios need to consolidate data from multiple sources, perform ETL/ELT operations, or maintain synchronized data warehouses. Essential for analytics, reporting, and ML scenarios that require clean, integrated datasets.
- **Technical requirements:**
  - Docker-based deployment (self-hosted or cloud)
  - PostgreSQL for metadata storage
  - Temporal for workflow orchestration
  - No-code Connector Builder for custom sources
  - CDK (Connector Development Kit) for advanced integrations
- **Key connectors to prioritize:**
  - Database: PostgreSQL, MySQL, MongoDB, Redis
  - APIs: REST, GraphQL, webhooks
  - Analytics: Qdrant, QuestDB
  - Files: CSV, JSON, Parquet from S3/MinIO
- **Integration points:** Orchestrate via n8n/Huginn, trigger workflows on sync completion, output to existing Vrooli data stores (PostgreSQL, Qdrant, MinIO)
- **Resource usage:** Medium; scales with data volume and connector complexity

#### **DeepStack** (Computer Vision)
- **What it does:** Cross-platform AI engine providing pre-built computer vision models via REST API. Includes object detection (80+ classes), face detection/recognition, scene recognition, and custom model training capabilities.
- **Use when:** Scenarios need computer vision without ML expertise - security monitoring, content moderation, automated tagging, accessibility features, or visual automation workflows.
- **Technical requirements:**
  - Docker deployment with CPU or GPU acceleration
  - REST API on configurable port (avoid hardcoding)
  - Redis for caching recognition results
  - Custom model support via training API
  - Integration with webcam feeds and image uploads
- **Key capabilities to expose:**
  - Object detection API for automation triggers
  - Face recognition for access control scenarios
  - Scene analysis for content categorization
  - Custom model training for domain-specific detection
- **Integration points:** Browserless screenshot analysis, n8n image processing workflows, Huginn monitoring agents, content moderation for generated apps
- **Resource usage:** Medium-heavy; GPU acceleration recommended for real-time processing

#### **FreeCAD** (Parametric CAD)
- **What it does:** Open-source parametric 3D CAD modeler with Python API for programmatic design generation. Supports architectural modeling, mechanical design, and engineering simulations with extensive file format compatibility.
- **Use when:** Scenarios need automated CAD generation, parametric design workflows, or integration between design and manufacturing. Complements existing OpenSCAD for more complex engineering workflows.
- **Technical requirements:**
  - Python 3.7+ with extensive scientific libraries
  - OpenGL for 3D rendering
  - Assembly workbench for multi-part designs
  - FEM workbench for finite element analysis
  - Path workbench for CAM/CNC integration
- **Key workbenches to enable:**
  - Part/PartDesign for solid modeling
  - Arch for architectural workflows
  - TechDraw for technical documentation
  - Assembly for multi-component designs
  - Path for manufacturing workflows
- **Integration points:** Export to 3D printers via OctoPrint, generate designs from n8n workflows, version control via git integration
- **Resource usage:** Medium; GPU beneficial for complex 3D rendering

#### **PyBullet** (Physics Simulation)
- **What it does:** Python-based physics simulation engine for robotics, reinforcement learning, and multi-physics simulations. Provides forward/inverse dynamics, collision detection, and VR/rendering capabilities.
- **Use when:** Scenarios need physics-based simulation, robotics testing, game physics, or reinforcement learning environments. Essential for validating robotic behaviors before real-world deployment.
- **Technical requirements:**
  - Python 3.7+ with NumPy/SciPy stack
  - OpenGL for visualization
  - URDF/SDF model loading support
  - Integration with ML frameworks (TensorFlow, PyTorch)
  - Real-time and headless execution modes
- **Key features to expose:**
  - Robot simulation with URDF models
  - Multi-object physics environments
  - Collision detection and ray tracing
  - VR headset integration for immersive simulation
  - RL gym environments for agent training
- **Integration points:** ROS2 bridge for robotics workflows, export results to data analysis scenarios, trigger simulations via n8n workflows
- **Resource usage:** Medium-heavy; benefits from GPU for rendering and parallel simulations

#### **OctoPrint** (3D Printing)
- **What it does:** Web-based 3D printer management platform with REST API for remote control, monitoring, and automation. Supports webcam streaming, print job management, and extensive plugin ecosystem.
- **Use when:** Scenarios involve 3D printing automation, print farm management, or integration of physical manufacturing into digital workflows. Essential for maker spaces and automated production scenarios.
- **Technical requirements:**
  - Python 3.7+ with Flask web framework
  - Serial communication for printer control
  - Webcam integration for monitoring
  - File management for G-code uploads
  - Plugin system for extensibility
- **Key plugins to pre-install:**
  - PrintWatch for AI-powered failure detection
  - OctoPrint-Telegram for notifications
  - Bed Leveling Assistant for automation
  - Multi-printer support plugins
- **Integration points:** Trigger prints from CAD scenarios, send notifications via existing communication channels, integrate with inventory management
- **Resource usage:** Low-medium; scales with number of connected printers and webcams

#### **Zigbee2MQTT** (IoT Bridge)
- **What it does:** Open-source Zigbee to MQTT bridge supporting 3000+ devices from various manufacturers. Eliminates proprietary hubs and enables local control of Zigbee smart home devices.
- **Use when:** Home automation scenarios need reliable, local control of Zigbee devices without cloud dependencies. Essential for privacy-focused smart home implementations.
- **Technical requirements:**
  - Zigbee USB adapter (CC2531, CC2652, or similar)
  - MQTT broker integration (existing infrastructure)
  - Node.js runtime environment
  - Device database for automatic discovery
  - Home Assistant integration via MQTT discovery
- **Key device categories to support:**
  - Sensors (temperature, motion, door/window)
  - Switches and dimmers
  - Smart plugs and relays
  - Environmental monitors
- **Integration points:** MQTT messages to n8n/Huginn workflows, Home Assistant automation, data logging to QuestDB
- **Resource usage:** Low; primarily I/O bound with minimal CPU requirements

#### **ESPHome** (IoT Firmware)
- **What it does:** Firmware framework for ESP32/ESP8266 microcontrollers using YAML configuration. Creates custom IoT devices with Home Assistant integration, OTA updates, and extensive sensor support.
- **Use when:** Scenarios need custom IoT sensors, actuators, or edge computing devices. Perfect for creating bespoke automation hardware integrated with existing smart home infrastructure.
- **Technical requirements:**
  - ESP32/ESP8266 development environment
  - YAML configuration management
  - Home Assistant integration
  - OTA update infrastructure
  - Sensor/actuator library ecosystem
- **Key components to support:**
  - Environmental sensors (temp, humidity, air quality)
  - Actuators (relays, servos, LEDs)
  - Communication (WiFi, Bluetooth, LoRa)
  - Display integration (OLED, e-ink)
- **Integration points:** Home Assistant device discovery, MQTT messaging, data visualization dashboards
- **Resource usage:** Very low; edge devices with minimal server-side requirements

#### **LNbits** (Lightning Payments)
- **What it does:** Open-source Lightning Network wallet and accounts system built on Python/FastAPI. Provides Bitcoin micropayments, API access, and extensible payment solutions with NOSTR integration.
- **Use when:** Scenarios need micropayments, Bitcoin integration, or decentralized payment systems. Essential for monetization, pay-per-use features, or crypto-native applications.
- **Technical requirements:**
  - Lightning Network node connection (LND, CLN, or others)
  - FastAPI Python web framework
  - SQLite/PostgreSQL for wallet data
  - NOSTR protocol integration
  - LNURL payment standards support
- **Key extensions to enable:**
  - Pay Links for static payment QRs
  - Vouchers for gift cards and credits
  - NOSTR integration for decentralized identity
  - Bolt cards for NFC payments
- **Integration points:** Payment verification in generated apps, micropayment triggers for n8n workflows, revenue tracking in analytics
- **Resource usage:** Low-medium; depends on Lightning node requirements

#### **mcrcon** (Minecraft Automation)
- **What it does:** Remote console client for Minecraft servers enabling programmatic command execution, server management, and player interaction automation. Supports both CLI and Python library interfaces.
- **Use when:** Building interactive virtual workspaces in Minecraft, educational scenarios, or creative team collaboration environments. Perfect for unconventional interfaces to Vrooli capabilities.
- **Technical requirements:**
  - Minecraft server with RCON enabled
  - Network connectivity to server
  - Command scripting capabilities
  - Player management and world manipulation
  - Integration with external data sources
- **Key automation capabilities:**
  - Player welcome and management systems
  - World state synchronization with external data
  - Interactive command execution from web interfaces
  - Scheduled maintenance and cleanup tasks
- **Integration points:** Web dashboards for world management, n8n workflows for automated events, data visualization using Minecraft blocks and structures
- **Resource usage:** Very low; primarily network I/O with minimal processing requirements

#### **JupyterHub** (Interactive Computing)
- **What it does:** Multi-user server that spawns, manages, and proxies multiple instances of Jupyter notebook servers. Enables collaborative data science, interactive computing, and educational environments with isolated workspaces for each user.
- **Use when:** Scenarios need interactive notebooks for data analysis, ML experimentation, documentation with live code, or educational workshops. Essential for data science teams and research scenarios requiring reproducible computing environments.
- **Technical requirements:**
  - Linux-based server (Windows not supported)
  - Python 3.7+ runtime environment
  - Docker/Kubernetes for container-based spawning
  - OAuth/GitHub/Google authentication providers
  - PostgreSQL for user database (production)
- **Key deployment patterns:**
  - Zero to JupyterHub for Kubernetes scalability
  - The Littlest JupyterHub for small deployments
  - Docker spawner for container isolation
  - Custom spawners for specific infrastructure
- **Integration points:** Git for version control, existing authentication systems, shared file systems via MinIO/NFS, connection to data sources (PostgreSQL, Qdrant), ML frameworks integration
- **Resource usage:** Scales with users; CPU/memory limits configurable per user; idle culling reduces resource waste

#### **Prometheus + Grafana** (Observability Stack)
- **What it does:** Complete monitoring and observability stack with Prometheus for metrics collection/alerting and Grafana for visualization. Provides real-time insights into resource health, performance, and usage patterns across the entire Vrooli platform.
- **Use when:** Need comprehensive monitoring of all resources, performance optimization, alerting on anomalies, or creating operational dashboards. Critical for maintaining platform reliability and enabling self-improvement feedback loops.
- **Technical requirements:**
  - Prometheus server for time-series data storage
  - Exporters for each monitored resource
  - Alertmanager for notification routing
  - Grafana for dashboard creation
  - Service discovery for dynamic targets
- **Key monitoring targets:**
  - All resource health metrics (CPU, memory, disk)
  - Application-specific metrics (request rates, error rates)
  - Business metrics (workflow completions, API usage)
  - Infrastructure metrics (network, containers)
- **Integration points:** Export to QuestDB for long-term storage, trigger n8n workflows from alerts, feed metrics to AI for anomaly detection, integrate with existing notification channels
- **Resource usage:** Medium; Prometheus storage scales with retention and cardinality; Grafana lightweight unless heavy dashboard usage

#### **Temporal** (Workflow Orchestration)
- **What it does:** Durable execution platform for building highly reliable distributed applications. Provides state management, automatic retries, and exactly-once execution guarantees for complex, long-running workflows that can span days or months.
- **Use when:** Scenarios require mission-critical workflows, multi-step processes with guaranteed completion, or coordination between multiple services. Superior to n8n for workflows requiring strong consistency, complex error handling, or very long execution times.
- **Technical requirements:**
  - Temporal Server (self-hosted or cloud)
  - PostgreSQL or Cassandra for persistence
  - Elasticsearch for visibility (optional)
  - SDK support (Go, Java, TypeScript, Python, PHP)
  - Worker processes for workflow execution
- **Key capabilities:**
  - Workflows that survive failures and resume exactly where they left off
  - Automatic retry policies with exponential backoff
  - Workflow versioning for zero-downtime updates
  - Signals and queries for runtime interaction
  - Child workflows and activity orchestration
- **Integration points:** Trigger from n8n for critical paths, orchestrate multi-agent AI workflows, manage distributed transactions, coordinate microservices, handle payment processing workflows
- **Resource usage:** Medium-heavy; scales with workflow complexity and throughput; requires dedicated workers

#### **WireGuard** (Secure Networking)
- **What it does:** Modern, lightweight VPN protocol providing secure point-to-point connections with state-of-the-art cryptography. Creates encrypted tunnels between distributed resources, enabling secure communication across untrusted networks.
- **Use when:** Resources need secure communication across different networks, remote access to local resources, or creating isolated network namespaces for containers. Essential for distributed Vrooli deployments or secure remote management.
- **Technical requirements:**
  - Linux kernel module or userspace implementation
  - Public key infrastructure setup
  - IP forwarding and NAT configuration
  - Firewall rules for port 51820 (default)
  - Network namespace support for isolation
- **Key security features:**
  - ChaCha20 encryption with Poly1305 authentication
  - Curve25519 for key exchange
  - Perfect Forward Secrecy with key rotation
  - Minimal attack surface (< 4000 lines of code)
  - Cryptokey routing for access control
- **Integration points:** Secure Docker container networking, cross-region resource connectivity, remote developer access, backup tunnel for critical services, mesh networking between nodes
- **Resource usage:** Very low; kernel-level operation minimizes overhead; handles gigabit speeds with minimal CPU

#### **Nextcloud** (File Collaboration)
- **What it does:** Self-hosted file sync and share platform with integrated office suite, video conferencing, and groupware. Provides complete collaboration infrastructure while maintaining full data sovereignty and privacy.
- **Use when:** Scenarios need file sharing, document collaboration, or team communication without third-party data exposure. Complements MinIO with user-friendly interfaces and rich collaboration features beyond raw object storage.
- **Technical requirements:**
  - PHP 8.0+ with Apache/Nginx
  - PostgreSQL/MySQL database
  - Redis for caching and file locking
  - Storage backend (local, S3-compatible, or external)
  - HTTPS with valid certificates
- **Key collaboration features:**
  - Nextcloud Office for document editing
  - Talk for video conferencing and chat
  - Groupware (Calendar, Contacts, Mail)
  - File versioning and sharing controls
  - End-to-end encryption options
- **Integration points:** MinIO as storage backend, LDAP/OAuth for authentication, n8n workflows via WebDAV API, mobile and desktop client sync, integration with existing office tools
- **Resource usage:** Medium; scales with active users and storage; benefits from SSD for database and cache

#### **Apache Kafka** (Event Streaming)
- **What it does:** Industry-standard distributed event streaming platform for high-throughput, fault-tolerant, and scalable messaging. Acts as a durable, high-speed pipeline between data producers and consumers, enabling real-time data processing and event-driven architectures.
- **Use when:** Scenarios need real-time event streaming, message queuing at scale, log aggregation, or building event-driven microservices. Essential for scenarios requiring guaranteed message delivery, stream processing, or event sourcing patterns.
- **Technical requirements:**
  - Java 8+ runtime environment
  - KRaft mode (3.3+) eliminates Zookeeper dependency
  - Native Docker image (apache/kafka) for easy deployment
  - Persistent storage for log segments
  - Network configuration for broker communication
- **Key capabilities:**
  - Topics and partitions for message organization
  - Consumer groups for parallel processing
  - Exactly-once semantics and transactions
  - Stream processing with Kafka Streams
  - Connect framework for data integration
  - Tiered storage for long-term event retention
- **Integration points:** Source/sink connectors for databases, MQTT bridge for IoT, n8n/Huginn event triggers, Airbyte data pipelines, real-time analytics with QuestDB, application integration via client libraries
- **Resource usage:** Medium-heavy; memory and disk I/O intensive; scales horizontally with broker count

#### **Apache Airflow** (Data Pipeline Orchestration)
- **What it does:** Python-based platform for programmatically authoring, scheduling, and monitoring data workflows as DAGs (Directed Acyclic Graphs). Version 3.0 (2025) brings DAG versioning, enhanced backfills, and client-server architecture for multi-cloud deployments.
- **Use when:** Building complex ETL/ELT pipelines, orchestrating data workflows across multiple systems, scheduling batch processing jobs, or managing dependencies between data tasks. Superior to n8n for data-centric workflows requiring Python logic.
- **Technical requirements:**
  - Python 3.8+ runtime environment
  - PostgreSQL for metadata storage
  - Redis/RabbitMQ for executor communication
  - Docker/Kubernetes for task isolation
  - Webserver and scheduler components
- **Key features (v3.0):**
  - DAG versioning for safe updates
  - Task Execution Interface for multi-language support
  - Edge Executor for IoT/edge deployments
  - Built-in backfill management
  - Rich operator ecosystem (1000+ integrations)
  - GitOps-friendly declarative configuration
- **Integration points:** Trigger from n8n for hybrid workflows, orchestrate Airbyte syncs, manage Kafka topics, coordinate ML training pipelines, integrate with cloud services, export metrics to monitoring systems
- **Resource usage:** Medium; scales with DAG complexity and parallelism; scheduler requires consistent resources

#### **Traefik** (API Gateway & Routing)
- **What it does:** Cloud-native reverse proxy and load balancer with automatic service discovery, dynamic configuration, and built-in observability. Simplifies routing, load balancing, and SSL management across all Vrooli resources without manual configuration.
- **Use when:** Need unified API gateway for all resources, automatic SSL certificates via Let's Encrypt, service mesh capabilities, or advanced routing rules. Essential for production deployments requiring zero-downtime updates and multi-cloud support.
- **Technical requirements:**
  - Single binary or Docker container
  - Supports Docker, Kubernetes, Consul service discovery
  - Minimal CPU/memory footprint
  - Network access to backend services
  - Optional persistent storage for certificates
- **Key capabilities:**
  - Automatic service discovery and configuration
  - Layer 4 and Layer 7 load balancing
  - Circuit breakers and rate limiting
  - Canary deployments and traffic mirroring
  - WebSocket and gRPC support
  - Kubernetes Gateway API v1.2.1 compliance
- **Integration points:** Route to all Vrooli resources, integrate with authentication providers, export metrics to Prometheus, trigger webhooks to n8n, manage certificates for HTTPS endpoints, API versioning and staging
- **Resource usage:** Very low; efficiently handles thousands of routes with minimal overhead

#### **Strapi** (Headless CMS)
- **What it does:** Open-source headless CMS built on Node.js that provides a customizable API and admin panel for content management. Version 5 offers 100% JavaScript/TypeScript codebase with REST and GraphQL APIs out of the box.
- **Use when:** Scenarios need structured content management, rapid API development, or multi-channel content delivery. Perfect for building content-driven applications without custom backend development.
- **Technical requirements:**
  - Node.js 18 or 20
  - PostgreSQL/MySQL/SQLite database
  - Docker support via official images
  - File storage (local or S3-compatible)
  - PM2 or similar for production
- **Key features (v5):**
  - Visual content type builder
  - Role-based access control (RBAC)
  - Internationalization (i18n) support
  - Media library with image optimization
  - Plugin ecosystem for extensibility
  - TypeScript support throughout
- **Integration points:** Store media in MinIO, trigger n8n workflows on content changes, authenticate via OAuth/SSO, serve content to any frontend framework, integrate with CDNs for global delivery, webhook notifications for content events
- **Resource usage:** Low-medium; Node.js efficiency with configurable scaling; database size grows with content

#### **Restic** (Backup & Recovery)
- **What it does:** Fast, secure backup program with client-side encryption, efficient deduplication, and support for multiple storage backends. Provides ransomware-resistant backups with temporary immutability and incremental snapshots.
- **Use when:** Need automated backup strategy for all Vrooli resources, disaster recovery capabilities, or cross-platform backup solution. Critical for protecting generated scenarios, databases, and configuration.
- **Technical requirements:**
  - Single Go binary (no dependencies)
  - Supports Linux, macOS, Windows, BSD
  - Storage backend (S3, MinIO, SFTP, local)
  - Minimal memory for operation
  - Network access to backup destination
- **Key capabilities:**
  - AES-256-GCM encryption before upload
  - Content-defined chunking for deduplication
  - Incremental forever with synthetic fulls
  - Snapshot-based recovery points
  - Parallel upload/download for speed
  - Resumable backups on connection loss
- **Integration points:** Back up to MinIO for S3-compatible storage, scheduled via cron/systemd, trigger from n8n workflows, integrate with monitoring for backup verification, Docker volume backups, PostgreSQL dump integration
- **Resource usage:** Very low during idle; CPU/network intensive during backup; deduplication reduces storage needs by 60%+

#### **Pi-hole** (Network-wide Ad Blocking & DNS Management)
- **What it does:** Network-level DNS sinkhole that blocks ads, tracking, and malware domains for all devices on the network. Provides centralized DNS management with a web interface and REST API for automation. Version 6 (2025) integrates API directly into FTL binary.
- **Use when:** Need network-wide ad blocking without per-device configuration, programmatic DNS control for home automation, or creating "clean" network environments for specific activities (movie nights, focused work sessions, etc.). Perfect for scenarios needing DNS-based filtering and monitoring.
- **Technical requirements:**
  - Docker container or native Linux installation
  - Port 53 (DNS) and 80/443 (web interface)
  - Static IP or reliable DHCP reservation
  - 512MB RAM minimum, 2-4GB disk space
  - Network configuration to use Pi-hole as DNS server
- **Key capabilities:**
  - Block ads at DNS level for all network devices
  - Custom blocklists and allowlists management
  - REST API v6 with session-based authentication
  - Gravity database with millions of blocked domains
  - Local DNS records for internal services
  - Query logging and statistics dashboard
  - Regex blocking for pattern matching
  - DNS-over-HTTPS (DoH) and DNS-over-TLS (DoT) support
- **Integration points:** Home Assistant for automation triggers, n8n workflows to enable/disable blocking on schedule, API endpoints for temporary disable during streaming, integration with network monitoring tools, custom blocklist updates from threat feeds, DNS resolution for internal Vrooli resources
- **API automation examples:**
  - Movie mode: Disable blocking for 2 hours via `/api/dns/blocking` endpoint
  - Kids' bedtime: Enable strict filtering after certain hours
  - Work hours: Apply productivity-focused blocklists during work time
  - Guest network: Different filtering policies per network segment
- **Resource usage:** Very low; typically 50-100MB RAM with minimal CPU; DNS caching reduces upstream queries

### 💡 Selection Priority
1. Fix non-running enabled resources
2. Add baseline content (Ollama models, n8n workflows)
3. Fix misconfigured connections
4. Add new resources only when 75%+ healthy

---

### 📎 Appendix — Command Patterns
- **Status/health**:
  - `vrooli resource status` (all resources)
  - `vrooli resource <name> status` (specific resource)
  - `resource-<name> status` (resource-specific CLI)
- **Start/Stop**:
  - `vrooli resource start <name>`
  - `vrooli resource stop <name>` (avoid stopping; Whisper post-test only, if needed)
- **Install/Improve**:
  - `vrooli resource install <name>`
- **Content Management** (replaces inject):
  - `resource-n8n content add --file workflow.json`
  - `resource-postgres content add --file schema.sql`
  - `resource-<name> content list`
  - `resource-<name> content execute --name my-workflow`
- **Sanity checks**:
  - Prefer resource CLIs or shared workflows (avoid direct client binaries unless explicitly allowed) 
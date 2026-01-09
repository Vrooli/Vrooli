<div align="center">

[
    <img alt="Vrooli logo with motto" src="./assets/readme-display.png" width="500px"/>
][website]


<h1>

[Website][website] | [Docs][docs] | [AI expert][chatGptBot]

</h1>

# Your Personal Software Development Server

**Imagine having your own AI development team that works entirely on your hardware.**

Vrooli is the first platform that lets you generate complete applications locally - UI, API, database, CLI - without sending your data to the cloud or depending on external services. Tell it what you want to build, and it creates working software in hours using 30+ local services running on your machine.

**🔒 Your Code. Your Data. Your Hardware. Your Control.**

</div>

## 🚀 Quick Start

```bash
# First time setup (installs CLI, configures resources)
make setup

# Start development environment
make dev

# Run tests
make test

# See all commands
make help
```

**Alternative commands:**
```bash
./scripts/manage.sh setup   # Direct setup script
vrooli develop             # Use CLI after setup
vrooli --help              # See all CLI commands
```

<div align="center">

<table style="width: 100%; table-layout: fixed;">
  <tr style="display: flex; flex-wrap: wrap; gap: 8px; justify-content: center;">
    <td align="center" style="flex: 0 1 auto;">
      <a href="https://vrooli.com" style="text-decoration: none; display: inline-block; white-space: nowrap;">
        <div style="display: inline-flex; align-items: center; background-color: #072c6a; color: #fff; border-radius: 12px; padding: 4px 8px; font-family: Arial, sans-serif; font-size: 14px; height: 30px;">
          <div style="background: white; border-radius: 50%; width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; margin-right: 6px; flex-shrink: 0;">
            <img src="https://www.google.com/s2/favicons?domain=vrooli.com&sz=16" width="16" height="16" />
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis;">Vrooli.com</span>
        </div>
      </a>
    </td>
    <td align="center" style="flex: 0 1 auto;">
      <a href="https://github.com/Vrooli/Vrooli" style="text-decoration: none; display: inline-block; white-space: nowrap;">
        <div style="display: inline-flex; align-items: center; background-color: #333; color: #fff; border-radius: 12px; padding: 4px 8px; font-family: Arial, sans-serif; font-size: 14px; height: 30px;">
          <div style="background: white; border-radius: 50%; width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; margin-right: 6px; flex-shrink: 0;">
            <img src="https://www.google.com/s2/favicons?domain=github.com&sz=16" width="16" height="16" />
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis;">Star Our Repo</span>
        </div>
      </a>
    </td>
    <td align="center" style="flex: 0 1 auto;">
      <a href="https://x.com/intent/follow?original_referer=https%3A%2F%2Fgithub.com%2FVrooliOfficial&screen_name=VrooliOfficial" style="text-decoration: none; display: inline-block; white-space: nowrap;">
        <div style="display: inline-flex; align-items: center; background-color: #111; color: #fff; border-radius: 12px; padding: 4px 8px; font-family: Arial, sans-serif; font-size: 14px; height: 30px;">
          <div style="background: white; border-radius: 50%; width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; margin-right: 6px; flex-shrink: 0;">
            <img src="https://www.google.com/s2/favicons?domain=x.com&sz=16" width="16" height="16" />
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis;">Follow @VrooliOfficial</span>
        </div>
      </a>
    </td>
    <td align="center" style="flex: 0 1 auto;">
      <a href="https://www.youtube.com/@vrooli" style="text-decoration: none; display: inline-block; white-space: nowrap;">
        <div style="display: inline-flex; align-items: center; background-color: #f00; color: #fff; border-radius: 12px; padding: 4px 8px; font-family: Arial, sans-serif; font-size: 14px; height: 30px;">
          <div style="background: white; border-radius: 50%; width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; margin-right: 6px; flex-shrink: 0;">
            <img src="https://www.google.com/s2/favicons?domain=youtube.com&sz=16" width="16" height="16" />
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis;">Subscribe @Vrooli</span>
        </div>
      </a>
    </td>
    <td align="center" style="flex: 0 1 auto;">
      <a href="https://choosealicense.com/licenses/agpl-3.0/" style="text-decoration: none; display: inline-block; white-space: nowrap;">
        <div style="display: inline-flex; align-items: center; background-color: #2a9d8f; color: #fff; border-radius: 12px; padding: 4px 8px; font-family: Arial, sans-serif; font-size: 14px; height: 30px;">
          <div style="background: white; border-radius: 50%; width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; margin-right: 6px; flex-shrink: 0;">
            <img src="https://www.google.com/s2/favicons?domain=choosealicense.com&sz=16" width="16" height="16" />
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis;">License: AGPLv3</span>
        </div>
      </a>
    </td>
  </tr>
</table>



<br/>

# Table of Contents
- [Why Vrooli Changes Everything](#why-vrooli-changes-everything)
- [How It Works](#how-it-works)
- [The Future of Software Development](#the-future-of-software-development)
- [Installation](#installation)
- [Quick Tutorial](#quick-tutorial)
- [Development](#development)
- [Contributing](#contributing)
- [Privacy & Data](#privacy--data)
- [License](#license)


## Why Vrooli Changes Everything

### 🏠 **True Privacy & Control**
- Everything runs on YOUR hardware - databases, AI models, applications
- Zero external API dependencies after initial setup
- Your proprietary business logic never leaves your network
- Perfect for sensitive industries (healthcare, finance, defense, research)
- Fully customizable security and compliance to your exact requirements

### 🚀 **AI Development Team in a Box**
- Generate complete applications: web UI + REST API + CLI + database
- 30+ integrated local services (databases, AI models, automation, storage)
- Modify AI-generated code safely - changes are preserved permanently
- Compose applications from small, focused scenarios (few thousand lines each)

### 🔧 **Modular Application Building**
Your applications aren't monolithic. They're built from composable scenarios:
- **Core scenarios**: Generate base app (UI + API + database + CLI)
- **Enhancement scenarios**: Add features (ui-component-manager, account-manager, branding-manager)
- **Platform scenarios**: Deploy and customize (app-to-ios, deployment-manager, app-issue-tracker)
- **Meta-scenarios**: The system improves itself (ecosystem-manager, system-monitor)

### 💰 **Real Business Value**
The applications Vrooli generates aren't demos - they're production-ready tools that businesses typically pay $10K-50K to develop. Whether for internal use or client delivery, you're getting genuine enterprise-grade software.

## How It Works

1. **Tell Vrooli What You Want**: Describe your application in plain English
2. **AI Orchestrates Local Resources**: Combines databases, workflows, UI frameworks automatically
3. **Complete Application Generated**: Working software with UI, API, CLI, and database
4. **Customize and Enhance**: Modify the code - your changes are automatically detected and preserved
5. **Compose Additional Features**: Add scenarios for iOS deployment, branding, debugging, monitoring

The genius is in the **scenario modification system**: AI generates your initial application, but when you customize it, those changes are detected and preserved. Future scenario updates won't overwrite your modifications.

**Current Resource Categories:** Storage, Automation, AI Models, Databases, UI Frameworks, Development Tools
**Coming Soon:** Home Automation, Physics Simulation, 3D Printing, CAD Integration, IoT Management

## The Future of Software Development

**Our vision: Every household and business running their own Vrooli server.**

Instead of depending on cloud services and external APIs, imagine:
- Your smart home running custom automation built by your personal AI
- Your business generating internal tools instantly without external developers
- Your family's personal assistant, photo manager, and productivity apps - all private
- Your company's sensitive processes automated without data ever leaving your premises
- Healthcare providers processing patient data with complete privacy
- Financial institutions running proprietary algorithms on their own hardware

We're building the infrastructure for a **locally-sovereign digital future**.

[Learn more about our business server solutions →](docs/business-solutions.md)


# Installation

## 💾 Local Server (Recommended)
Run Vrooli on your own infrastructure with complete control and privacy:

```bash
# Quick setup with default resources (includes Ollama AI models)
./scripts/manage.sh setup

# Start your personal development server
vrooli develop
```

**Why Local?** This unlocks Vrooli's full potential:
- **Complete Privacy:** Your code, data, and AI models never leave your hardware
- **Full Resource Access:** 30+ local services for building complete applications
- **No External Dependencies:** Generate applications even when offline
- **Enterprise Ready:** Perfect for sensitive business environments

[See detailed setup guide →][setup-guide]

## 🌐 Hosted Service (Alternative)
Prefer not to manage your own server? We also offer a hosted version at [vrooli.com](https://vrooli.com) with privacy-focused features:
- Your data stays in your chosen region
- No tracking or advertising  
- Full data export/deletion controls
- Option to migrate to local deployment anytime

[Private hosting guide →][private-hosting]


# Quick Tutorial

Video coming soon! In the meantime, new accounts will be greeted with an interactive tutorial that will guide you through:

1. **First Application**: Generate a personal task manager with UI, API, and CLI
2. **Customization**: Modify the generated code to fit your needs
3. **Enhancement**: Add features using additional scenarios
4. **Privacy Setup**: Configure local-only operation

Example scenarios you'll learn to build:
- Personal productivity tools
- Business automation dashboards
- Research and analysis applications
- Custom workflows and integrations


# Development

Vrooli uses a modern, privacy-first technology stack designed for local resource orchestration and AI-driven application generation.

### Core Technologies
- **React + TypeScript:** Type-safe frontend with real-time AI interaction
- **Node.js + Express:** High-performance backend for local resource coordination  
- **PostgreSQL + pgvector:** Local database with AI embedding support
- **Docker:** Containerized local services for security and isolation
- **Redis:** Local caching and real-time coordination

### Architecture Highlights
- **Local-First Design:** Everything runs on your hardware
- **Scenario-Based:** Modular, composable application templates
- **Resource Orchestration:** 30+ integrated local services
- **AI Model Agnostic:** Works with OpenAI, Anthropic, Mistral, local models
- **Privacy by Design:** No external dependencies after setup

[Detailed architecture documentation →](docs/ARCHITECTURE_OVERVIEW.md)

## Privacy & Security

Vrooli's local-first architecture provides inherent security advantages:

### Privacy by Design
- **Local Execution:** All data processing happens on your hardware
- **No Cloud Dependencies:** Generate applications completely offline
- **Configurable Security:** Add encryption, compliance, monitoring as needed
- **Industry Ready:** GDPR, HIPAA, SOX compliance capabilities

### Security Features
- **Sandboxed Execution:** Isolated containers for safe code generation
- **Permission Controls:** Granular access based on roles and requirements
- **Audit Trails:** Comprehensive logging for compliance and forensics
- **Emergency Controls:** Immediate system halt for safety-critical situations

[Complete security documentation →](docs/security/README.md)

## [🗂️ Project Structure][project-structure]
[This docs section][project-structure] provides a comprehensive overview of the project's structure, helping developers get familiar with the layout and organization of the codebase.

## [👩🏼‍💻 Developer Setup][setup-guide]
[Follow this guide][setup-guide] to set up your development environment, including step-by-step instructions and useful tips for efficient development.

**Note:** Running the setup process with sudo permissions automatically configures any required system settings for local resources.

### CI/CD Pipeline

We have set up a CI/CD pipeline to automatically deploy changes to a development VPS whenever changes are pushed to the `development` branch. This allows for quick testing and validation of changes before they are merged into the main branch and deployed to production.

For detailed instructions on how to set up and use the CI/CD pipeline, see the [CI/CD Setup documentation](docs/devops/ci-cd.md).

# Contributing

## Multilingual Support
Vrooli is building the future of local-first software development. We welcome contributions from developers who share our vision of digital sovereignty.

### How to Contribute
- **Scenario Development:** Create new application templates for the community
- **Resource Integration:** Add support for new local services and tools  
- **Platform Development:** Core platform improvements and optimizations
- **Documentation:** Help others understand and use Vrooli effectively
- **Translation:** Make Vrooli accessible in multiple languages

### Current Priorities
- Local resource integrations (home automation, 3D printing, IoT)
- Enterprise scenario templates (healthcare, finance, manufacturing)
- Mobile and desktop application deployment scenarios
- Community scenario marketplace development

## Join the Team
### Get Started
1. Check our [project board](https://github.com/orgs/Vrooli/projects/1) for open tasks
2. Read the [development guide](docs/development/README.md)  
3. Join our community discussions
4. Submit pull requests for review

As we become profitable, we'll add bounty rewards for completed contributions. This is your chance to shape the future of local-first development.

[Contact the maintainer →](https://matthalloran.info)


## Privacy & Data

**Vrooli is designed with privacy as the foundation, not an afterthought.**

### Local-First Privacy
- **Your Hardware, Your Data:** When running locally, all data stays on your machine
- **No Cloud Dependencies:** Generate applications completely offline
- **Zero Tracking:** No analytics, ads, or external data collection
- **Complete Control:** You own and control every piece of data

### Hosted Service Privacy
If you use our hosted service:
- **Minimal Data Collection:** Only what's necessary for operation
- **Clear Data Boundaries:** Always know what's private vs. public
- **Full Export/Deletion:** Complete control over your data
- **No Third-Party Sharing:** Your data is never sold or shared
- **Compliance Ready:** GDPR and CCPA compliant

### Enterprise & Sensitive Data
Perfect for industries requiring strict data control:
- **Healthcare:** HIPAA-compliant local deployment
- **Finance:** Regulatory compliance with local data processing
- **Defense:** Air-gapped environments supported
- **Research:** Proprietary algorithms stay completely private

[Complete privacy policy →](https://vrooli.com/privacy)


# License

Vrooli is released under the [GNU Affero General Public License v3.0 (AGPLv3)][license].

## Why AGPL 3.0?

- **Freedom and Transparency:** Ensures all improvements remain open source, even for network services
- **Community Growth:** Encourages collaborative development and shared innovation
- **Anti-Vendor Lock-in:** Prevents proprietary capture of community contributions
- **Local Sovereignty:** Supports the vision of personal, local-first computing

The AGPL ensures that Vrooli will always remain free software that empowers users, not vendors.


### [🌍**Let's change the world together!🕊️**][website]


[website]: https://vrooli.com
[docs]: https://docs.vrooli.com
[chatGptBot]: https://chatgpt.com/g/g-WbecuwZSy-vrooli-product-manager
[personal-site]: https://matthalloran.info
[setup-guide]: https://github.com/MattHalloran/ReactGraphQLTemplate#how-to-start
[project-structure]: https://docs.vrooli.com/setup/project_structure.html
[private-hosting]: https://docs.vrooli.com/setup/getting_started/remote_setup.html
[x]: https://x.com/intent/follow?original_referer=https%3A%2F%2Fgithub.com%2FVrooliOfficial&screen_name=VrooliOfficial
[youtube]: https://www.youtube.com/@vrooli
[email]: mailto:support@vrooli.com
[github]: https://github.com/Vrooli/Vrooli
[license]: https://choosealicense.com/licenses/agpl-3.0/
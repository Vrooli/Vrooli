# Legacy Full-Stack Kubernetes Artifacts

This directory preserves the old Vrooli full-stack Kubernetes deployment artifacts as historical reference.

These files reflect the pre-tiered deployment model where Vrooli was treated as one deployable platform stack rather than a set of individually deployable scenarios.

Do not treat this directory as a supported deployment path.

Use it only for:

- historical context
- mining older Helm/Vault/operator assumptions
- future `deployment-manager` or `scenario-to-cloud` research

Current deployment truth lives in:

- `docs/deployment/README.md`
- `docs/operations/production-guide.md`
- `scenarios/deployment-manager/docs/`

Archived items here include:

- `k8s/`: legacy Helm chart, values, manifests, and support files
- `github-workflows/k8s-deploy.yml`: retired GitHub Actions workflow preserved for reference only

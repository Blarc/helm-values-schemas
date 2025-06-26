# Helm Values Schema Generator

A lightweight HTTP service that generates JSON schema for Helm charts. The service downloads values.yaml files from Helm chart repositories, generates schema using the losisin/helm-values-schema-json library, and returns the schema as JSON.

## Example
Official Loki Helm Chart lacks a JSON schema: https://github.com/grafana/loki/tree/main/production/helm/loki.

Generate one using: https://helm-values-schemas.onrender.com/grafana/loki/refs/heads/main/production/helm/loki/values.yaml

**URL Pattern:** Take any GitHub raw URL like:

https://raw.githubusercontent.com/grafana/loki/refs/heads/main/production/helm/loki/values.yaml

Replace `https://raw.githubusercontent.com` with the service URL:

https://helm-values-schemas.onrender.com/grafana/loki/refs/heads/main/production/helm/loki/values.yaml

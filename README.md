# kubebuilder-admission-poc

One CRD, one reconciler, and both admission webhooks served by the **same controller-runtime
Manager**. No second binary, no second Deployment, no hand-registered webhook configuration.

## Why

KubeEdge currently runs reconciliation through controller-runtime and admission through a separate
command with its own HTTP server, its own TLS setup, and webhook configurations built in Go at
runtime. I'm applying to consolidate those, so I built the end state at a size I could finish.

## Status

Nothing works yet.

# kubebuilder-admission-poc

One CRD, one reconciler, and both admission webhooks served by the **same controller-runtime
Manager**. One binary, one Deployment, webhook configuration generated from markers rather than
built in Go at runtime.

Kubebuilder v4, controller-runtime v0.20, Kubernetes 1.32, Go 1.25.12.

## Why

I'm applying to consolidate KubeEdge's admission controller into its controller manager, so I built
the end state at a size I could finish. I have not contributed to KubeEdge before. This exists to
show that I understand the problem and can implement it, not to propose an answer.

[docs/design.md](docs/design.md) has the mapping and the parts that are easy to get wrong.

## Run it

Needs Go and Docker.

```
make test     # unit + envtest, against a real API server
make demo     # deploy to a throwaway kind cluster and exercise admission
make help
```

## What it does

`Device` has a model, a protocol, an endpoint and a node.

- **Defaulting webhook** fills in `protocol` (mqtt) and `offlineMigration` (true).
- **Validating webhook** rejects an endpoint whose scheme doesn't match its protocol, makes `model`
  immutable, and refuses a node change unless `offlineMigration` is on.
- **Reconciler** sets `observedGeneration` and a `Ready` condition.

The CRD enum already covers "is this a known protocol". The webhook is there for the things a schema
cannot see: the previous object, and the relationship between two fields.

## The bits worth reading

**`cmd/main.go`** — the whole point. `WebhookServer` is a field on `ctrl.Options`, so the webhooks
inherit the Manager's client, cache, scheme, logger, probes, metrics and leader election. Readiness
is gated on `GetWebhookServer().StartedChecker()`, because a pod that goes Ready before it can serve
admission makes the API server reject every write with `failurePolicy: Fail`.

**`internal/webhook/v1alpha1/suite_test.go`** — envtest installs the generated webhook manifests and
points a real API server at the Manager. Calling `Default()` directly would prove the function works
and nothing about whether the configuration reaches it, which is where this kind of migration
actually breaks.

**`hack/verify-go-version.sh`** — fails the build when `.go-version`, `go.mod` and the Dockerfile
disagree, and when a workflow pins Go literally instead of using `go-version-file: go.mod`. Drift
here is quiet: a patched runtime in one place and not another.

**`Makefile`** — `controller-gen` and `setup-envtest` are pinned and installed into `./bin`, never
taken from `PATH`. A generator version that depends on the developer is a generated-file diff
waiting to happen.

## Toolchain

`make vulncheck` was not decoration. The first run found 25 reachable advisories: two in
`golang.org/x/net`, one in `golang.org/x/text`, and the rest in the standard library via
`crypto/x509`, `crypto/tls`, `net/textproto`, `mime` and `net`. Fixed by bumping the two modules and
moving to Go 1.25.12. It reports clean now, and CI fails if that stops being true.

## Not here

One CRD instead of six. No conversion webhooks, no migration path from an existing deployment, which
in the real thing is most of the work.
